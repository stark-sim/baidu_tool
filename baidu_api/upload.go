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
	// 该信道控制上传文件信息
	uploadFileInfoChan := make(chan *FileInfo)
	// 该信道控制创建文件信息
	createFileInfoChan := make(chan *FileInfo)

	// 使用了多个协程在高层逻辑，需要一个信号来关闭大家当 panic 级别错误出现或结束
	closeChan := make(chan struct{})

	// 做预创建文件的协程
	go func() {
		// 使用协程进行对多个文件并行预上传
		preCreateWG := &sync.WaitGroup{}
		for _, localFilePath := range localFilePaths {
			// 对于一个本地文件，至少要预上传一个文件
			preCreateWG.Add(1)

			// 第一步，检查文件是否超过 20GB
			fileInfo, err := os.Stat(localFilePath)
			if err != nil {
				close(closeChan)
				return
			}
			fileSize := fileInfo.Size()
			// 超级会员单文件限制
			if fileSize > utils.MaxSingleFileSize {
				// 分成的文件数量
				var fileNum int
				if fileSize%utils.MaxSingleFileSize != 0 {
					// 无法整除，就会有最后一个不足 20GB 的文件
					fileNum = 1
				}
				fileNum += int(fileSize / utils.MaxSingleFileSize)
				// 比预期的单文件要额外多预上传 fileNum - 1 个文件
				preCreateWG.Add(fileNum - 1)
				for i := 1; i <= fileNum; i++ {
					// 准备开始预上传，需要网络
					limitChan <- struct{}{}
					time.Sleep(time.Second + time.Millisecond*time.Duration(rand.Intn(100)))
					go func(index int, bigLocalFilePath string) {
						var tempFileInfo FileInfo
						tempFileInfo.PreCreateReturn, tempFileInfo.BaiduFilePath, tempFileInfo.BlockList, tempFileInfo.FileSize, err = upload.PreCreate(accessToken, bigLocalFilePath, baiduPrefixPath, index)
						if err != nil {
							log.Printf("%v\n", err)
							close(closeChan)
							return
						}

						// 预上传接口调用成功后，这个文件接下来会被开始上传，与此同时就该启动 文件切片传输字节信道 来呼应接下来的上传
						tempFileInfo.SlicedFileBytesChan = make(chan *utils.SlicedFileByte)
						// 预上传部分占用并行数量必须在 影响上传部分 之前释放
						<-limitChan
						// 当前该文件信息已经完成好上传前所有准备工作，可以推送给上传文件信道
						uploadFileInfoChan <- &tempFileInfo
						preCreateWG.Done()
						// 释放一个并行量
						// 推送到上传信道好后，可以开始传输切片
						if err = utils.SliceFilePushToChan(bigLocalFilePath, tempFileInfo.SlicedFileBytesChan, index, tempFileInfo.FileSize); err != nil {
							log.Printf("sliceFilePushToChan err: %v", err)
							close(closeChan)
						}
					}(i, localFilePath)
				}
			} else {
				// 单文件开始预上传
				limitChan <- struct{}{}
				go func(singleLocalFilePath string) {
					var preFileInfo FileInfo
					preFileInfo.PreCreateReturn, preFileInfo.BaiduFilePath, preFileInfo.BlockList, preFileInfo.FileSize, err = upload.PreCreate(accessToken, singleLocalFilePath, baiduPrefixPath, 0)
					if err != nil {
						log.Printf("%v\n", err)
						close(closeChan)
						return
					}
					preFileInfo.SlicedFileBytesChan = make(chan *utils.SlicedFileByte)
					// 预上传部分占用并行数量必须在 影响上传部分 之前释放
					<-limitChan
					uploadFileInfoChan <- &preFileInfo
					preCreateWG.Done()
					if err := utils.SliceFilePushToChan(singleLocalFilePath, preFileInfo.SlicedFileBytesChan, 0, 0); err != nil {
						log.Printf("err: %+v", err)
						close(closeChan)
					}
				}(localFilePath)
			}
		}
		preCreateWG.Wait()
		// 需要往 uploadChannel 输送的数据已经输送好了，可以关闭 uploadChannel
		close(uploadFileInfoChan)
	}()

	// 做上传碎片文件的协程
	go func() {
		// 上传步骤为主要步骤，要控制文件上传的顺序，避免第一个碎片文件和最后一个碎片文件直接间隔太长
		uploadLimitChan := make(chan struct{}, 2)
		// 所有上传文件的 wg
		uploadWG := &sync.WaitGroup{}
		for {
			select {
			case <-closeChan:
				return
			case uploadFileInfo, ok := <-uploadFileInfoChan:
				if !ok {
					// upload 信道已关闭，等自己要 upload 的事情做完就可以关闭 create 信道
					uploadWG.Wait()
					close(createFileInfoChan)
					return
				}
				// 多一个要上传的文件
				uploadWG.Add(1)
				// 为该文件争取到并发量
				uploadLimitChan <- struct{}{}
				// 上传开启，新建一个进度条
				uploadFileInfo.Bar = progress.AddBar(
					uploadFileInfo.FileSize,
					mpb.PrependDecorators(decor.Name(uploadFileInfo.BaiduFilePath), decor.Percentage(decor.WCSyncSpace)),
					mpb.BarRemoveOnComplete(),
				)
				go func(fileInfo *FileInfo) {
					// 该文件的碎片上传 wg 同步控制，
					slicedUploadWaitGroup := &sync.WaitGroup{}
					for slicedFileByte := range fileInfo.SlicedFileBytesChan {
						limitChan <- struct{}{}
						time.Sleep(time.Second + time.Millisecond*time.Duration(rand.Intn(100)))
						slicedUploadWaitGroup.Add(1)
						go func(fileBytes *utils.SlicedFileByte, smallFileInfo *FileInfo) {
							if _, err := upload.SingleUpload(accessToken, smallFileInfo.PreCreateReturn.UploadId, smallFileInfo.BaiduFilePath, fileBytes.Bytes, fileBytes.Index); err != nil {
								close(closeChan)
								return
							}
							<-limitChan
							smallFileInfo.Bar.IncrBy(int(utils.ChunkSize))
							slicedUploadWaitGroup.Done()
						}(slicedFileByte, fileInfo)
					}
					slicedUploadWaitGroup.Wait()
					// 文件的上传过程完成，推送信息到最后的创建文件信道
					createFileInfoChan <- fileInfo
					// 一个整文件的完成
					uploadWG.Done()
					<-uploadLimitChan
				}(uploadFileInfo)
			}
		}
	}()

	// 做收尾创建文件的协程
	go func() {
		createWG := &sync.WaitGroup{}
		for {
			select {
			case <-closeChan:
				return
			case createFileInfo, ok := <-createFileInfoChan:
				if !ok {
					// 已经没有新的要创建的文件了
					// 等最后一个文件创建完毕后，就可以全局关闭
					createWG.Wait()
					close(closeChan)
					return
				}
				createWG.Add(1)
				limitChan <- struct{}{}
				time.Sleep(time.Second + time.Millisecond*time.Duration(rand.Intn(100)))
				go func(fileInfo *FileInfo) {
					_, err := upload.Create(accessToken, fileInfo.BaiduFilePath, fileInfo.FileSize, fileInfo.BlockList, fileInfo.PreCreateReturn.UploadId)
					if err != nil {
						log.Printf("err: %v\n", err)
						close(closeChan)
					}
					<-limitChan
					// 创建完表示一个文件处理完毕
					createWG.Done()
				}(createFileInfo)
			}
		}
	}()

	// 同步收尾
	select {
	case <-closeChan:
		// END
	}
	return nil
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
