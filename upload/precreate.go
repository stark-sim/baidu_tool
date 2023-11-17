package upload

import (
	"baidu_tool/utils"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type PreCreateArg struct {
	Path      string   `json:"path"`
	Size      uint64   `json:"size"`
	BlockList []string `json:"block_list"`
}

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
	Size uint64 `json:"size"`
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

func PreCreate(accessToken string, arg *PreCreateArg) (*PreCreateReturn, error) {
	ret := &PreCreateReturn{}

	client := http.Client{}

	uri := "https://pan.baidu.com/rest/2.0/xpan/file?method=precreate&access_token=%s"
	realUrl, _ := url.Parse(fmt.Sprintf(uri, accessToken))

	jsonBody, err := json.Marshal(PreCreateBody{
		Path:      arg.Path,
		Size:      arg.Size,
		IsDir:     0,
		BlockList: arg.BlockList,
		AutoInit:  1,
		RType:     1,
	})
	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("User-Agent", "pan.baidu.com")
	req := &http.Request{
		Method: "Post",
		URL:    realUrl,
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(jsonBody)),
	}

	ret, err = utils.DoHttpRequest(ret, &client, req)
	if err != nil {
		return nil, err
	}
	if ret.Errno != 0 {
		return nil, errors.New("call pre_create failed")
	}
	return ret, nil
}
