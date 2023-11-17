package utils

import (
	"baidu_tool/baidu_api"
	"crypto/md5"
	"fmt"
	"log"
	"os"
)

const MaxSingleFileSize int64 = 20 * 1024 * 1024 * 1024
const ChunkSize int64 = 4 * 1024 * 1024

func SliceFile(localFilePath string) (slicedFilePaths []string, blockList []string, err error) {
	// Try to read the file
	file, err := os.Open(localFilePath)
	if err != nil {
		log.Fatal("error trying to open the file specified:", err)
	}
	defer file.Close()

	dir, _, err := baidu_api.DivideDirAndFile(localFilePath)
	if err != nil {
		return nil, nil, err
	}

	// Check if the provided directory path for partials exits; create if it doesn't
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Fatal("error creating directory:", err)
			return nil, nil, err
		}
	}

	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		return nil, nil, err
	}
	fullFileSize := fileInfo.Size()
	sliceFileNum := fullFileSize / ChunkSize
	lastSize := fullFileSize % ChunkSize
	// 不能整除，意味着还有一个碎文件
	if lastSize != 0 {
		sliceFileNum++
	}

	// 文件坑位为分快大小
	b := make([]byte, ChunkSize)
	for i := int64(0); i < sliceFileNum; i++ {

		//_, err := file.Seek(i*ChunkSize, 0)
		//if err != nil {
		//	log.Fatal("error seek the file:", err)
		//	return err
		//}

		// 最后一个文件的 bytes 坑位切换成大小
		if i == sliceFileNum-1 {
			b = make([]byte, lastSize)
		}

		// 读取分块字节
		if _, err = file.Read(b); err != nil {
			return nil, nil, err
		}

		sliceFileName := fmt.Sprintf("%s%d", localFilePath, i)
		// 切分时记录文件的信息
		slicedFilePaths = append(slicedFilePaths, sliceFileName)
		blockList = append(blockList, fmt.Sprintf("%x", md5.Sum(b)))

		f, err := os.OpenFile(sliceFileName, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return nil, nil, err
		}
		_, err = f.Write(b)
		if err != nil {
			log.Fatal("error write a file:", err)
			return nil, nil, err
		}
		err = f.Close()
		if err != nil {
			log.Fatal("error close a file:", err)
			return nil, nil, err
		}
	}

	return slicedFilePaths, blockList, nil
}
