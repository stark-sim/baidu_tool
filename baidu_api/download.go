package baidu_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type DownloadLinkResp struct {
	Errmsg    string          `json:"errmsg"`
	Errno     int8            `json:"errno"`
	List      []*DownloadInfo `json:"list"`
	RequestID string          `json:"request_id"`
}

type DownloadInfo struct {
	Category int8   `json:"category"`
	DLink    string `json:"dlink"`
	Filename string `json:"filename"`
	FsID     int64  `json:"fs_id"`
	MD5      string `json:"md5"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

type fileIndexPath struct {
	FilePath string
	Index    int
}

const MB50 = 20 * 1024 * 1024

func DownloadFileOrDir(accessToken string, source []*FileOrDir) error {
	//FsIDMapDLink := make(map[int64]*DownloadInfo)
	var fsIDList []int64
	for _, item := range source {
		// 下载一个文件
		if item.IsDir == 1 {
			continue
		} else {
			// 先收集 fs_id
			fsIDList = append(fsIDList, item.FsId)
		}
	}

	// 用 fs_id 换取下载地址
	downloadInfos, err := getDownloadInfo(accessToken, fsIDList)
	if err != nil {
		return err
	}

	// 拿到下载地址后，开始协程下载
	// 协程下载最高并发，cpu 数量
	limitChan := make(chan struct{}, runtime.NumCPU())
	defer close(limitChan)
	client := http.Client{}
	// 整理文件结果的协程要有信号量来知道全都处理好了，主协程才能结束
	joinSliceWG := &sync.WaitGroup{}
	for _, downloadInfo := range downloadInfos {
		// 如果文件太大，就下切片
		var lastSize int64
		var sliceNum int64
		if downloadInfo.Size > MB50 {
			// 多少个 50 MB
			sliceNum = downloadInfo.Size / MB50
			// 下载完最后的碎片多大
			lastSize = downloadInfo.Size % MB50
		} else {
			lastSize = downloadInfo.Size
		}

		// 准备下载请求
		_url := downloadInfo.DLink + "&access_token=" + accessToken
		realUrl, err := url.Parse(_url)
		if err != nil {
			return err
		}
		if sliceNum > 0 {
			// 请先看非协程部分代码，只有 limitChan 会起到代码阻塞作用，其他的下载，结果拼接过程都是在协程中进行的。
			// 目的是为了充分发挥网络并发能力，可以让多个文件同时以切片形式下载

			// 每一个分片下载的文件都有一个信道作为最后收集碎片文件信息的媒介
			tempFileChan := make(chan *fileIndexPath, 5)
			// 有一个独立协程做收集文件信息并最后拼接操作
			joinSliceWG.Add(1)
			go func(innerFileChan chan *fileIndexPath, finalFileName string) {
				var sliceFileIndexPaths []*fileIndexPath
				for tempFileIndexPath := range innerFileChan {
					sliceFileIndexPaths = append(sliceFileIndexPaths, tempFileIndexPath)
				}
				targetFile, err := os.OpenFile(finalFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
				defer targetFile.Close()
				if err != nil {
					fmt.Printf("打开目标文件错误: %v\n", err)
					return
				}
				// 按升序排序
				sort.Slice(sliceFileIndexPaths, func(i, j int) bool {
					return sliceFileIndexPaths[i].Index < sliceFileIndexPaths[j].Index
				})
				for i := 0; i < len(sliceFileIndexPaths); i++ {
					content, err := os.ReadFile(sliceFileIndexPaths[i].FilePath)
					if err != nil {
						fmt.Printf("读碎片文件错误")
						return
					}
					_, err = targetFile.Write(content)
					if err != nil {
						fmt.Printf("追加文件错误")
						return
					}
					go func(sliceFile string) {
						if err = os.Remove(sliceFile); err != nil {
							fmt.Printf("删除碎片文件错误")
							return
						}
					}(sliceFileIndexPaths[i].FilePath)
				}

				fmt.Printf("文件拼接好了 %s\n", finalFileName)
				joinSliceWG.Done()
			}(tempFileChan, "."+downloadInfo.Path)

			// 分片下载需要一个信号量让接受文件结果协程知道收集可以结束
			downloadWG := &sync.WaitGroup{}
			// 分片下载带拼接
			for i := 0; i < int(sliceNum); i++ {
				limitChan <- struct{}{}
				// 下载部分启动协程下载，启动协程受限于并发控制信道
				downloadWG.Add(1)
				go func(sliceIndex int, innerFileChan chan *fileIndexPath, innerDownloadWG *sync.WaitGroup, _url *url.URL, baiduFilePath string) {
					header := http.Header{}
					header.Set("User-Agent", "pan.baidu.com")
					header.Set("Range", fmt.Sprintf("bytes=%v-%v", sliceIndex*MB50, sliceIndex*MB50+MB50-1))
					request := http.Request{
						Method: "GET",
						URL:    _url,
						Header: header,
					}
					// 重传直到完成
					for {
						resp, err := client.Do(&request)
						if err != nil {
							fmt.Printf("网络连接错误 clientDo\n")
							continue
						}
						if resp.StatusCode != 206 {
							fmt.Printf("状态码非 206\n")
							continue
						}
						defer resp.Body.Close()
						respBytes, err := io.ReadAll(resp.Body)
						if err != nil {
							fmt.Printf("返回读取错误 ioReadAll\n")
							continue
						}
						// 网络请求下载好后要收回下载并发信号量
						<-limitChan
						// 把文件保存为碎片文件
						localDownloadFilePath := fmt.Sprintf(".%s%d", baiduFilePath, sliceIndex)
						dir, _, err := DivideDirAndFile(localDownloadFilePath)
						if err != nil {
							continue
						}
						if err = os.MkdirAll(dir, 0750); err != nil {
							fmt.Printf("创建文件夹错误 mkdirAll\n")
							continue
						}
						if err = os.WriteFile(localDownloadFilePath, respBytes, 0666); err != nil {
							fmt.Printf("写文件错误 osWriteFile\n")
							continue
						}
						// 保存好文件后，推送自己完成的文件信息
						innerFileChan <- &fileIndexPath{
							FilePath: localDownloadFilePath,
							Index:    sliceIndex,
						}
						innerDownloadWG.Done()
						break
					}
				}(i, tempFileChan, downloadWG, realUrl, downloadInfo.Path)
			}
			// 再下载最后一个文件，也是要控制并发地下载
			limitChan <- struct{}{}
			downloadWG.Add(1)
			go func(innerFileChan chan *fileIndexPath, innerDownloadWG *sync.WaitGroup, _url *url.URL, baiduFilePath string) {
				header := http.Header{}
				header.Set("User-Agent", "pan.baidu.com")
				header.Set("Range", fmt.Sprintf("bytes=%v-%v", sliceNum*MB50, sliceNum*MB50+lastSize-1))
				request := http.Request{
					Method: "GET",
					URL:    _url,
					Header: header,
				}
				// 重传直到通过
				for {
					resp, err := client.Do(&request)
					if err != nil {
						fmt.Printf("网络连接错误 clientDo\n")
						continue
					}
					if resp.StatusCode != 206 {
						fmt.Printf("状态码非 206\n")
						continue
					}
					defer resp.Body.Close()
					respBytes, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Printf("返回读取错误 ioReadAll\n")
						continue
					}
					<-limitChan
					// 把文件保存为碎片文件
					localDownloadFilePath := fmt.Sprintf(".%s%d", baiduFilePath, sliceNum)
					dir, _, err := DivideDirAndFile(localDownloadFilePath)
					if err != nil {
						continue
					}
					if err = os.MkdirAll(dir, 0750); err != nil {
						fmt.Printf("创建文件夹错误 mkdirAll\n")
						continue
					}
					if err = os.WriteFile(localDownloadFilePath, respBytes, 0666); err != nil {
						fmt.Printf("写文件错误 osWriteFile\n")
						continue
					}
					// 保存好文件后，推送自己完成的文件信息
					innerFileChan <- &fileIndexPath{
						FilePath: localDownloadFilePath,
						Index:    int(sliceNum),
					}
					innerDownloadWG.Done()
					break
				}
			}(tempFileChan, downloadWG, realUrl, downloadInfo.Path)

			// 这一步也不阻塞，因为还有下一个文件
			go func(innerDownloadWG *sync.WaitGroup, innerChan chan *fileIndexPath) {
				// 都下载并传输结果完毕
				innerDownloadWG.Wait()
				// 那么就可以关闭结果传输信道，让结果收集者知道收集完了
				close(innerChan)
			}(downloadWG, tempFileChan)

		} else {
			// 不分片，直接下
			limitChan <- struct{}{}
			go func(_url *url.URL, baiduFilePath string) {
				header := http.Header{}
				header.Set("User-Agent", "pan.baidu.com")
				request := http.Request{
					Method: "GET",
					URL:    _url,
					Header: header,
				}
				for {
					resp, err := client.Do(&request)
					if err != nil {
						fmt.Printf("网络连接错误 clientDo\n")
						continue
					}
					if resp.StatusCode == 206 {
						fmt.Printf("状态码非 206 \n")
						continue
					}
					defer resp.Body.Close()
					respBytes, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Printf("返回读取错误 ioReadAll\n")
						continue
					}
					<-limitChan
					// 把文件保存为一个文件
					localDownloadFilePath := "." + baiduFilePath
					dir, _, err := DivideDirAndFile(localDownloadFilePath)
					if err != nil {
						continue
					}
					if err = os.MkdirAll(dir, 0750); err != nil {
						fmt.Printf("创建文件夹错误 mkdirAll\n")
						continue
					}
					if err = os.WriteFile(localDownloadFilePath, respBytes, 0666); err != nil {
						fmt.Printf("写文件错误 osWriteFile\n")
						continue
					}
					break
				}
			}(realUrl, downloadInfo.Path)
		}
	}
	// 这个 wg 结束了，那就都结束了
	joinSliceWG.Wait()
	return nil
}

// 一次性拿到要下载的文件的下载地址们
func getDownloadInfo(accessToken string, fsIDList []int64) ([]*DownloadInfo, error) {
	if fsIDList == nil || len(fsIDList) == 0 {
		return nil, nil
	}
	preUrl := "http://pan.baidu.com/rest/2.0/xpan/multimedia?method=filemetas&access_token=%s&fsids=%s&dlink=1"
	var strFsIDList []string
	for _, fsID := range fsIDList {
		strFsIDList = append(strFsIDList, strconv.FormatInt(fsID, 10))
	}
	fsids := "[" + strings.Join(strFsIDList, ",") + "]"
	resp, err := http.Get(fmt.Sprintf(preUrl, accessToken, fsids))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var downloadResp DownloadLinkResp
	if err = json.Unmarshal(respBytes, &downloadResp); err != nil {
		return nil, err
	}
	if downloadResp.Errno != 0 {
		return nil, fmt.Errorf("api no return or return err: %v", downloadResp)
	}
	return downloadResp.List, nil
}
