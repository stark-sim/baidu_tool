package baidu_api

import (
	"baidu_tool/upload"
	"baidu_tool/utils"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"
)

type FileInfo struct {
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
	uploadFileInfoChan := make(chan *FileInfo, 2)
	createFileInfoChan := make(chan *FileInfo, 2)

	// 使用了多个协程在高层逻辑，需要一个信号来关闭大家当 panic 级别错误出现
	closeChan := make(chan struct{})

	// 做预创建文件的协程
	go func() {
		// 使用协程进行对多个文件并行预上传
		for _, localFilePath := range localFilePaths {
			go func(localFilePath string) {
				// 第一步，检查文件是否超过 20GB
				fileInfo, err := os.Stat(localFilePath)
				if err != nil {
					close(closeChan)
					return
				}
				fileSize := fileInfo.Size()
				// 超级会员单文件限制
				if fileSize > utils.MaxSingleFileSize {
					// 分成多个文件来上传
					var preFileInfoList []*FileInfo
					// 分成的文件数量
					var fileNum int
					if fileSize%utils.MaxSingleFileSize != 0 {
						// 无法整除，就会有最后一个不足 20GB 的文件
						fileNum = 1
					}
					fileNum += int(fileSize / utils.MaxSingleFileSize)
					for i := 1; i <= fileNum; i++ {
						var tempFileInfo FileInfo
						tempFileInfo.PreCreateReturn, tempFileInfo.BaiduFilePath, tempFileInfo.BlockList, tempFileInfo.FileSize, err = upload.PreCreate(accessToken, localFilePath, baiduPrefixPath, i)
						if err != nil {
							log.Printf("%v\n", err)
							close(closeChan)
							return
						}
						// 预上传后有了具体文件目标名称和大小，开启一个进度条
						tempFileInfo.Bar = progress.AddBar(
							tempFileInfo.FileSize,
							mpb.PrependDecorators(decor.Name(tempFileInfo.BaiduFilePath)),
							mpb.BarRemoveOnComplete(),
						)
						// 预上传接口调用成功后，这个文件接下来会被开始上传，与此同时就该启动 文件切片传输字节信道 来呼应接下来的上传
						tempFileInfo.SlicedFileBytesChan = make(chan *utils.SlicedFileByte)
						// 当前该文件信息已经完成好上传前所有准备工作，可以推送给上传文件信道
						uploadFileInfoChan <- &tempFileInfo
						// 推送好后，
					}
				} else {
					// 单文件开始预上传
					var preFileInfo FileInfo
					preFileInfo.PreCreateReturn, preFileInfo.BaiduFilePath, preFileInfo.BlockList, preFileInfo.FileSize, err = upload.PreCreate(accessToken, localFilePath, baiduPrefixPath, 0)
					if err != nil {
						log.Printf("%v\n", err)
						close(closeChan)
						return
					}
					// 预上传后有了文件大小，开启一个进度条
					preFileInfo.Bar = progress.AddBar(
						preFileInfo.FileSize,
						mpb.PrependDecorators(decor.Name(preFileInfo.BaiduFilePath)),
						mpb.BarRemoveOnComplete(),
					)
					preFileInfo.SlicedFileBytesChan = make(chan *utils.SlicedFileByte)
					preFileInfoChan <- &preFileInfo
					if err := utils.SliceFilePushToChan(localFilePath, preFileInfo.SlicedFileBytesChan, 0); err != nil {
						log.Printf("err: %+v", err)
						close(closeChan)
					}
				}

			}(localFilePath)
		}
	}()

	// 做上传碎片文件的协程
	go func() {

	}()

	// 做收尾创建文件的协程
	go func() {

	}()

	uploadWaitGroup := &sync.WaitGroup{}
	for {
		select {
		case <-closeChan:
			return nil

		case preFileInfo := <-preFileInfoChan:
			go func(preFileInfo *FileInfo) {
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
