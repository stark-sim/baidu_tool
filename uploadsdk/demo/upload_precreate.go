package main

import (
	"baidu_tool/uploadsdk/upload"
	"fmt"
)

func main() {
	// 使用示例

	// 用户的access_token
	accessToken := "123.1075c43cdb2e381bc039d5fb3591e777.YCDbbVlVn1o9lUnEAuv70JOG-ZlsGdw6mCP-F9n.3IMbnw"

	// 要保存到网盘的文件路径
	// 前缀必须为：/apps/您的应用名/
	path := "/apps/hhhkoo/testfile.pdf"

	// 测试文件testfile.pdf的大小，单位：B
	var size uint64
	size = 7141246

	// testfile.pdf 6.8MB，超出4MB。以4MB为单位切分，切分为两个：aa和ab。
	// 分片aa 的md5 为18f2607ce63714753ef640be507fe8f4
	// 分片ab 的md5 为f4bea7d615c2429258f615397c7cd11c
	// MD5是一种通用的加密算法
	// shell语言下，获取文件aa MD5的语法是： md5sum aa
	var blockList []string
	blockList = append(blockList, "18f2607ce63714753ef640be507fe8f4")
	blockList = append(blockList, "f4bea7d615c2429258f615397c7cd11c")

	// call precreate API
	arg := upload.NewPrecreateArg(path, size, blockList)
	if ret, err := upload.Precreate(accessToken, arg); err != nil {
		fmt.Printf("[msg: precreate error] [err:%v]", err.Error())
	} else {
		fmt.Printf("ret:%+v", ret)
	}
}
