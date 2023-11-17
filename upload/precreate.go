package upload

import (
	"baidu_tool/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type PreCreateReturn struct {
	Errno      int    `json:"errno"`
	ReturnType int    `json:"return_type"`
	BlockList  []int  `json:"block_list"` // 分片序号列表
	UploadId   string `json:"uploadid"`
	RequestId  int    `json:"request_id"`
}

type PreCreateBody struct {
	// 上传后使用的文件绝对路径，需要 url_encode
	Path string `json:"path"`
	// 文件和目录两种情况：上传文件时，表示文件的大小，单位 B；上传目录时，表示目录的大小，目录的话大小默认为 0
	Size int64 `json:"size"`
	// 否为目录，0 文件，1 目录
	IsDir int `json:"isdir"`
	// 文件各分片MD5数组的json串。
	// block_list 的含义如下，如果上传的文件小于 4MB，其 md5 值（32位小写）即为block_list字符串数组的唯一元素；
	// 如果上传的文件大于 4MB，需要将上传的文件按照 4MB 大小在本地切分成分片，不足 4MB 的分片自动成为最后一个分片，
	// 所有分片的 md5值（32位小写）组成的字符串数组即为 block_list。
	BlockList []string `json:"block_list"`
	// 固定值 1
	AutoInit int `json:"autoinit"`
	// 文件命名策略。
	// 1 表示当 path 冲突时，进行重命名
	// 2 表示当 path 冲突且 block_list 不同时，进行重命名
	// 3 当云端存在同名文件时，对该文件进行覆盖
	RType int `json:"rtype"`
}

// PreCreate 预上传
// @param localFilePath 上传后使用的文件绝对路径，需要 url encode
func PreCreate(accessToken string, localFilePath string) (*PreCreateReturn, []string, int64, error) {
	ret := &PreCreateReturn{}

	client := http.Client{}
	uri := "https://pan.baidu.com/rest/2.0/xpan/file?method=precreate&access_token=%s"
	realUrl, _ := url.Parse(fmt.Sprintf(uri, accessToken))

	// md5
	md5res, fileSize, err := utils.FileToMd5(localFilePath)
	if err != nil {
		return nil, nil, 0, err
	}

	// json body 参数
	body := url.Values{}
	body.Add("path", "/apps/"+localFilePath)
	body.Add("size", strconv.FormatInt(fileSize/8, 10))
	body.Add("isdir", "0")
	blockListJson, _ := json.Marshal([]string{md5res})
	body.Add("block_list", string(blockListJson))
	body.Add("autoinit", "1")
	body.Add("rtype", "2")

	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("User-Agent", "pan.baidu.com")
	req := &http.Request{
		Method: "POST",
		URL:    realUrl,
		Header: header,
		Body:   io.NopCloser(strings.NewReader(body.Encode())),
	}

	ret, err = utils.DoHttpRequest(ret, &client, req)
	if err != nil {
		return nil, nil, 0, err
	}
	if ret.Errno != 0 {
		return nil, nil, 0, errors.New("call pre_create failed")
	}
	return ret, []string{md5res}, fileSize, nil
}
