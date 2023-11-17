package utils

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"
)

func SliceFile(localFilePath string) {

	var (
		partsDir       = flag.String("d", "parts", "Provide a directory path where partial files will be stored")
		partFileSuffix = flag.String("s", "", "Provide a suffix for each part.")
		partsFileName  = flag.String("p", "part_", "Provide a prefix for each part.")
	)

	*partFileSuffix = path.Ext(localFilePath)
	fileNameAll := path.Base(localFilePath)
	*partsFileName = fileNameAll
	*partsDir = path.Dir(localFilePath)
	*partsDir = fmt.Sprintf("%s%s", *partsDir, "/parts")
	flag.Parse()

	// Try to read the file
	file, err := os.Open(localFilePath)
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		log.Fatal("error trying to open the file specified:", err)
	}
	defer file.Close()

	// Check if the provided directory path for partials exits; create if it doesn't
	if _, err := os.Stat(*partsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(*partsDir, os.ModePerm); err != nil {
			log.Fatal("error creating directory:", err)
			return
		}
	}

	var _ *os.File
	var fileName string

	if err != nil {
		log.Fatal("error trying to get the size:", err)
		return
	}
	chunkSize := 4 * 1024 * 1024
	num := int(math.Ceil(float64(fileInfo.Size()) / float64(chunkSize)))

	b := make([]byte, chunkSize)
	var i int64 = 1
	for ; i <= int64(num); i++ {

		_, err := file.Seek((i-1)*int64(chunkSize), 0)
		if err != nil {
			log.Fatal("error seek the file:", err)
			return
		}

		if len(b) > int((fileInfo.Size() - (i-1)*int64(chunkSize))) {
			b = make([]byte, fileInfo.Size()-(i-1)*int64(chunkSize))
		}

		file.Read(b)
		fileName = fmt.Sprintf("%s%s%d", fileNameAll, "_", i)
		fileName = filepath.Join(*partsDir, fileName)

		if _, err = os.Create(fileName); err != nil {
			log.Fatal("error creating a file:", err)
		}

		f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return
		}
		_, err = f.Write(b)
		if err != nil {
			log.Fatal("error write a file:", err)
			return
		}
		err = f.Close()
		if err != nil {
			log.Fatal("error close a file:", err)
			return
		}
	}

}
