package utils

import (
	"crypto/md5"
	"fmt"
	"os"
)

func FileToMd5(localFilePath string) (md5res string, err error) {
	f, err := os.OpenFile(localFilePath, os.O_RDONLY, 0755)
	if err != nil {
		return "", err
	}
	defer f.Close()
	// 获取文件大小
	fileInfo, err := f.Stat()
	if err != nil {
		return "", err
	}
	fileBts := make([]byte, fileInfo.Size())
	_, err = f.Read(fileBts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(fileBts)), nil
}
