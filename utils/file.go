package utils

import "os"

func SliceFile(localFilePath string) {
	file, err := os.OpenFile(localFilePath, os.O_RDONLY, 0755)

}
