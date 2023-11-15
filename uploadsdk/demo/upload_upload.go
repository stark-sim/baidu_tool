package main

import (
	"baidu_tool/uploadsdk/upload"
	"fmt"
)

func main() {
	// 使用示例

	// 用户的access_token
	accessToken := "your-access-token"

	// uploadId为precreate返回的uploadId
	uploadId := "P1-MTAuMTQ0LjEyOC4xNToxNjUxMDI5MTE1OjM5NTM0NjY0MTQ4NjIxMzE0OA=="

	// path与precreate的一致
	path := "/apps/hhhkoo/testfile.pdf"

	// 上传第1个分片
	file := "第1个分片aa的本地文件路径"
	partseq := 0
	// call upload API
	arg := upload.NewUploadArg(uploadId, path, file, partseq)
	if ret, err := upload.Upload(accessToken, arg); err != nil {
		fmt.Printf("[msg: upload this part error] [err:%v]", err.Error())
	} else {
		fmt.Printf("ret:%+v", ret)
	}

	// 上传第2个分片
	file = "第2个分片ab的本地文件路径"
	partseq = 1
	// call upload API
	arg = upload.NewUploadArg(uploadId, path, file, partseq)
	if ret, err := upload.Upload(accessToken, arg); err != nil {
		fmt.Printf("[msg: upload this part error] [err:%v]", err.Error())
	} else {
		fmt.Printf("ret:%+v", ret)
	}

}
