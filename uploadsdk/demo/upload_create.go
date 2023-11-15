package main

import (
	"baidu_tool/uploadsdk/upload"
	"fmt"
)

func main() {

	// 使用示例

	// 用户的access_token
	accessToken := "your-access-token"

	// path与precreate的一致
	path := "/apps/hhhkoo/testfile.pdf"

	// size与precreate的一致
	var size uint64
	size = 7141246

	// blocklist与precreate的一致
	blockList := []string{}
	blockList = append(blockList, "18f2607ce63714753ef640be507fe8f4")
	blockList = append(blockList, "f4bea7d615c2429258f615397c7cd11c")

	// uploadId为precreate返回的uploadId
	uploadId := "P1-MTAuMTQ0LjEyOC4xNToxNjUxMDI5MTE1OjM5NTM0NjY0MTQ4NjIxMzE0OA=="

	// // call create API
	arg := upload.NewCreateArg(uploadId, path, size, blockList)
	if ret, err := upload.Create(accessToken, arg); err != nil {
		fmt.Printf("[msg: create this part error] [err:%v]", err.Error())
	} else {
		fmt.Printf("ret:%+v", ret)
	}

}
