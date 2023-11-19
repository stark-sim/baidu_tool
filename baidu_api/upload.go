package baidu_api

import (
	"baidu_tool/upload"
	"baidu_tool/utils"
	"fmt"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

// UploadFileOrDir 上传文件或者文件夹
// @param localFilePath 要上传的文件或文件夹的相对位置或绝对位置
// @param baiduPrefixPath 上传后在网盘内的 我的应用数据/baiduPrefixPath/localFilePath 如果没传就在 我的应用数据/localFilePath
func UploadFileOrDir(accessToken string, localFilePath string, baiduPrefixPath string, progress *mpb.Progress) error {
	// 预上传，不保存文件
	preCreateReturn, baiduFilePath, blockList, fileSize, err := upload.PreCreate(accessToken, localFilePath, baiduPrefixPath)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	// 预上传后有了文件大小，开启一个进度条
	bar := progress.AddBar(
		fileSize,
		mpb.PrependDecorators(decor.Name(baiduFilePath)),
		mpb.BarRemoveOnComplete(),
	)

	// 上传过程，再次切文件，但这次最多同时保留进程数量的 字节段 在内存中（不需要保存文件）
	maxConcurrentCount := min(16, runtime.NumCPU())
	// 该信道控制上传协程并发量
	limitChan := make(chan struct{}, maxConcurrentCount)
	defer close(limitChan)
	// 该信道为切片文件字节传输信道，只需要单长度即可，不然没得传，切多了也浪费空间，由发送端关闭
	slicedFileBytesChan := make(chan *utils.SlicedFileByte)
	go func() {
		if err = utils.SliceFilePushToChan(localFilePath, slicedFileBytesChan); err != nil {
			return
		}
	}()

	// 一旦文件切完，文件传输信道就会关闭，但碎片上传还未结束，所以需要上传协程的同步量
	uploadWaitGroup := &sync.WaitGroup{}
	for slicedFileByte := range slicedFileBytesChan {
		limitChan <- struct{}{}
		// 协程上传时需要缓一缓，不然容易被百度关掉
		time.Sleep(time.Second + time.Millisecond*time.Duration(rand.Intn(100)))
		uploadWaitGroup.Add(1)
		go func(fileBytes *utils.SlicedFileByte) {
			if _, err = upload.SingleUpload(accessToken, preCreateReturn.UploadId, baiduFilePath, fileBytes.Bytes, fileBytes.Index); err != nil {
				return
			}
			<-limitChan
			bar.IncrBy(int(utils.ChunkSize))
			uploadWaitGroup.Done()
		}(slicedFileByte)
	}
	uploadWaitGroup.Wait()
	// 上传完成后，才可以进行 create 操作
	_, err = upload.Create(accessToken, baiduFilePath, fileSize, blockList, preCreateReturn.UploadId)
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
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
