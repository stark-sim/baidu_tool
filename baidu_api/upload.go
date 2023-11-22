package baidu_api

import (
	"baidu_tool/upload"
	"baidu_tool/utils"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type PreFileInfo struct {
	PreCreateReturn     *upload.PreCreateReturn
	BaiduFilePath       string
	BlockList           []string
	FileSize            int64
	SlicedFileBytesChan chan *utils.SlicedFileByte
	Bar                 *mpb.Bar
}

// UploadFileOrDir 上传文件或者文件夹
// @param localFilePath 要上传的文件或文件夹的相对位置或绝对位置
// @param baiduPrefixPath 上传后在网盘内的 我的应用数据/baiduPrefixPath/localFilePath 如果没传就在 我的应用数据/localFilePath
func UploadFileOrDir(accessToken string, localFilePaths []string, baiduPrefixPath string, progress *mpb.Progress) error {
	// 上传过程，再次切文件，但这次最多同时保留进程数量的 字节段 在内存中（不需要保存文件）
	maxConcurrentCount := min(16, runtime.NumCPU())
	// 该信道控制上传协程并发量
	limitChan := make(chan struct{}, maxConcurrentCount)
	defer close(limitChan)
	// 该管道控制预上传文件信息
	PreFileInfoChan := make(chan *PreFileInfo, 2)

	closeChan := make(chan struct{})

	// 使用协程进行文件预上传以及切片
	for _, localFilePath := range localFilePaths {
		go func(localFilePath string) {
			// 预上传，不保存文件
			var preFileInfo PreFileInfo
			var err error
			preFileInfo.PreCreateReturn, preFileInfo.BaiduFilePath, preFileInfo.BlockList, preFileInfo.FileSize, err = upload.PreCreate(accessToken, localFilePath, baiduPrefixPath)
			if err != nil {
				log.Printf("%v\n", err)
				close(closeChan)
			}
			// 预上传后有了文件大小，开启一个进度条
			preFileInfo.Bar = progress.AddBar(
				preFileInfo.FileSize,
				mpb.PrependDecorators(decor.Name(preFileInfo.BaiduFilePath)),
				mpb.BarRemoveOnComplete(),
			)
			preFileInfo.SlicedFileBytesChan = make(chan *utils.SlicedFileByte)
			PreFileInfoChan <- &preFileInfo
			if err := utils.SliceFilePushToChan(localFilePath, preFileInfo.SlicedFileBytesChan); err != nil {
				log.Printf("err: %+v", err)
				close(closeChan)
			}

		}(localFilePath)
	}
	uploadWaitGroup := &sync.WaitGroup{}
	var CreatedNum int
	for {
		select {
		case <-closeChan:
			return nil

		case preFileInfo := <-PreFileInfoChan:
			go func(preFileInfo *PreFileInfo) {
				for slicedFileByte := range preFileInfo.SlicedFileBytesChan {
					limitChan <- struct{}{}
					time.Sleep(time.Second + time.Millisecond*time.Duration(rand.Intn(100)))
					uploadWaitGroup.Add(1)
					go func(fileBytes *utils.SlicedFileByte) {
						if _, err := upload.SingleUpload(accessToken, preFileInfo.PreCreateReturn.UploadId, preFileInfo.BaiduFilePath, fileBytes.Bytes, fileBytes.Index); err != nil {
							close(closeChan)
							return
						}
						<-limitChan
						preFileInfo.Bar.IncrBy(int(utils.ChunkSize))
						uploadWaitGroup.Done()
					}(slicedFileByte)
				}
				uploadWaitGroup.Wait()
				_, err := upload.Create(accessToken, preFileInfo.BaiduFilePath, preFileInfo.FileSize, preFileInfo.BlockList, preFileInfo.PreCreateReturn.UploadId)
				if err != nil {
					log.Printf("err: %v\n", err)
					close(closeChan)
				}
				// fixme: 当所有的文件上传结束，那么就标志着传输结束，那么就结束程序
				CreatedNum++
				if CreatedNum == len(localFilePaths) {
					close(closeChan)
				}
			}(preFileInfo)
		}
	}

}

// ParseBaiduPrefixPath 处理传入的百度前缀地址，去除首尾可能存在的 '/'
func ParseBaiduPrefixPath(baiduPrefixPath string) string {
	if baiduPrefixPath == "" {
		return ""
	}
	// 如果有内容，那么去除掉头尾可能存在的 /
	if baiduPrefixPath[0] == '/' {
		baiduPrefixPath = baiduPrefixPath[1:]
	}
	if baiduPrefixPath[len(baiduPrefixPath)-1] == '/' {
		baiduPrefixPath = baiduPrefixPath[:len(baiduPrefixPath)-1]
	}
	return baiduPrefixPath
}
