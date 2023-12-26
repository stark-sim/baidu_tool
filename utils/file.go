package utils

import (
	"crypto/md5"
	"fmt"
	"log"
	"os"
	"strings"
)

const MaxSingleFileSize int64 = 20 * 1024 * 1024 * 1024
const ChunkSize int64 = 4 * 1024 * 1024

// GetFilePathListFromLocalPath 列表形式返回文件或者文件夹下所有文件的路径
func GetFilePathListFromLocalPath(localFileOrDirPath string) ([]string, error) {
	var filePathList []string
	fileInfo, err := os.Stat(localFileOrDirPath)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		children, err := os.ReadDir(localFileOrDirPath)
		if err != nil {
			return nil, err
		}
		for _, item := range children {
			childrenFilePathList, err := GetFilePathListFromLocalPath(localFileOrDirPath + "/" + item.Name())
			if err != nil {
				return nil, err
			}
			filePathList = append(filePathList, childrenFilePathList...)
		}
	} else {
		// 文件就进入结果
		filePathList = append(filePathList, localFileOrDirPath)
	}
	return filePathList, nil
}

type SlicedFileByte struct {
	Index int
	Bytes []byte
}

// SliceFilePushToChan 把文件一块块切割后推入给好的 channel，所以要使用协程来运行该函数
func SliceFilePushToChan(localFilePath string, slicedFileByteChan chan *SlicedFileByte, sequence int, fileSize int64) (err error) {
	// Try to read the file
	file, err := os.Open(localFilePath)
	if err != nil {
		log.Fatal("error trying to open the file specified:", err)
	}
	defer file.Close()

	var fullFileSize int64
	if sequence != 0 {
		fullFileSize = fileSize
		if _, err = file.Seek(int64(sequence-1)*MaxSingleFileSize, 0); err != nil {
			return err
		}
	} else {
		fileInfo, err := os.Stat(localFilePath)
		if err != nil {
			return err
		}
		fullFileSize = fileInfo.Size()
	}
	sliceFileNum := fullFileSize / ChunkSize
	lastSize := fullFileSize % ChunkSize
	// 不能整除，意味着还有一个碎文件
	if lastSize != 0 {
		sliceFileNum++
	}

	// 文件坑位为分块大小
	for i := int64(0); i < sliceFileNum; i++ {
		b := make([]byte, ChunkSize)
		// 最后一个文件的 bytes 坑位切换成大小
		if i == sliceFileNum-1 {
			b = make([]byte, lastSize)
		}

		// 读取分块字节
		if _, err = file.Read(b); err != nil {
			return
		}

		// 准备输出
		slicedFileByte := new(SlicedFileByte)
		slicedFileByte.Bytes = b
		slicedFileByte.Index = int(i)

		slicedFileByteChan <- slicedFileByte
	}

	// 发送完毕后，由发送端关闭信道
	close(slicedFileByteChan)

	return nil
}

// SliceFileNotSave 分片不保存，省空间，只提供 碎片文件的 md5 列表，为百度 preCreate 接口服务
func SliceFileNotSave(localFilePath string, sequence int, fileSize int64) (md5List []string, err error) {
	// Try to read the file
	file, err := os.Open(localFilePath)
	if err != nil {
		log.Fatal("error trying to open the file specified:", err)
	}
	defer file.Close()

	var fullFileSize int64
	// 只从 localPath 中读取部分文件
	if sequence != 0 {
		fullFileSize = fileSize
		// 使用 seek 调整，来决定读取第几个
		if _, err = file.Seek(int64(sequence-1)*MaxSingleFileSize, 0); err != nil {
			return nil, err
		}
	} else {
		fileInfo, err := os.Stat(localFilePath)
		if err != nil {
			return nil, err
		}
		fullFileSize = fileInfo.Size()
	}

	sliceFileNum := fullFileSize / ChunkSize
	lastSize := fullFileSize % ChunkSize
	// 不能整除，意味着还有一个碎文件
	if lastSize != 0 {
		sliceFileNum++
	}

	// 文件坑位为分快大小
	b := make([]byte, ChunkSize)
	for i := int64(0); i < sliceFileNum; i++ {
		// 最后一个文件的 bytes 坑位切换成大小
		if i == sliceFileNum-1 {
			b = make([]byte, lastSize)
		}

		// 读取分块字节
		if _, err = file.Read(b); err != nil {
			return
		}

		// 切分时记录文件的信息
		md5List = append(md5List, fmt.Sprintf("%x", md5.Sum(b)))
	}
	return
}

// SliceFileAndSave 分片并保存碎片文件方案
func SliceFileAndSave(localFilePath string) (slicedFilePaths []string, blockList []string, err error) {
	// Try to read the file
	file, err := os.Open(localFilePath)
	if err != nil {
		log.Fatal("error trying to open the file specified:", err)
	}
	defer file.Close()

	dir, _, err := DivideDirAndFile(localFilePath)
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

// DivideDirAndFile 分离文件路径中的 / 来找出文件夹和文件名
func DivideDirAndFile(filePath string) (dir string, file string, err error) {
	// 如果最后一个字符是 / ，则相当于最后一个 / 没有意义，可以去掉
	if filePath[len(filePath)-1] == '/' {
		filePath = filePath[:len(filePath)-1]
	}
	// 再开始从尾找第一个 /
	lastIndex := strings.LastIndex(filePath, "/")
	if lastIndex == -1 {
		return "", "", fmt.Errorf("not found /")
	}
	return filePath[:lastIndex], filePath[lastIndex+1:], nil
}
