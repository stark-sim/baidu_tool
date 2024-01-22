package utils

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const MaxSingleFileSize int64 = 1024 * 1024 * 1024
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

		// 准备输出
		slicedFileByte := new(SlicedFileByte)
		slicedFileByte.Index = int(i)

		slicedFileByte.Bytes = make([]byte, ChunkSize)
		// 最后一个文件的 bytes 坑位切换成大小
		if lastSize != 0 && i == sliceFileNum-1 {
			slicedFileByte.Bytes = make([]byte, lastSize)
		}

		// 读取分块字节
		if _, err = file.Read(slicedFileByte.Bytes); err != nil {
			log.Fatal(err)
			return
		}

		if len(slicedFileByte.Bytes) == 0 {
			fmt.Printf("%+v", slicedFileByte)
		}

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
		if lastSize != 0 && i == sliceFileNum-1 {
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
		if lastSize != 0 && i == sliceFileNum-1 {
			b = make([]byte, lastSize)
		}

		// 读取分块字节
		if _, err = file.Read(b); err != nil {
			return nil, nil, err
		}

		sliceFileName := fmt.Sprintf("%s_%d", localFilePath, i)
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

// JigsawSlicedFiles 合并本脚步拆分的超大文件碎片
func JigsawSlicedFiles(slicedFilesDir string) error {
	// 输入的是碎片文件所在的文件夹
	targetDirInfo, err := os.Stat(slicedFilesDir)
	if err != nil {
		return err
	}
	if !targetDirInfo.IsDir() {
		return errors.New("input is not a dir")
	}

	_, fileName, err := DivideDirAndFile(slicedFilesDir)
	if err != nil {
		return err
	}
	targetFile, err := os.OpenFile(filepath.Join(slicedFilesDir, fileName), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	defer targetFile.Close()
	if err != nil {
		fmt.Printf("打开目标文件错误: %v\n", err)
		return err
	}

	// 拼接后看情况删除文件，都删除完拼接过程再算结束
	//removeSliceWG := &sync.WaitGroup{}
	// 碎片文件名从 1 开始，按十进制递增
	for i := int64(1); i <= 1024; i++ {
		content, err := os.ReadFile(filepath.Join(slicedFilesDir, strconv.FormatInt(i, 10)))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// 碎片文件已遍历完，结束
				break
			}
			fmt.Printf("读碎片文件错误")
			return err
		}
		_, err = targetFile.Write(content)
		if err != nil {
			fmt.Printf("追加文件错误")
			return err
		}
		//removeSliceWG.Add(1)
		//go func(sliceFile string, wg *sync.WaitGroup) {
		//	if err = os.Remove(sliceFile); err != nil {
		//		fmt.Printf("删除碎片文件错误")
		//		return
		//	}
		//}(sliceFileIndexPaths[i].FilePath, removeSliceWG)
	}

	fmt.Printf("文件拼接好了 %s\n", fmt.Sprintf("%s/%s", slicedFilesDir, fileName))
	return nil
}
