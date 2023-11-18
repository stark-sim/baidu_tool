package upload

import (
	"baidu_tool/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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
func PreCreate(accessToken string, localFilePath string, prefixPath string) (ret *PreCreateReturn, baiduFilePath string, blockList []string, fileSize int64, err error) {
	// 准备返回体，第一步
	ret = &PreCreateReturn{}

	// 检查文件大小
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		return
	}
	fileSize = fileInfo.Size()
	// 超级会员单文件限制
	if fileSize > utils.MaxSingleFileSize {
		err = errors.New(fmt.Sprintf("file %s too big, not can do", localFilePath))
		return
	}

	if fileSize > utils.ChunkSize {
		// 需要分块
		blockList, err = utils.SliceFileNotSave(localFilePath)
		if err != nil {
			return
		}
	} else {
		// md5
		var md5Res string
		md5Res, err = utils.FileToMd5(localFilePath)
		if err != nil {
			return
		}
		blockList = append(blockList, md5Res)
	}

	client := http.Client{}
	uri := "https://pan.baidu.com/rest/2.0/xpan/file?method=precreate&access_token=%s"
	realUrl, _ := url.Parse(fmt.Sprintf(uri, accessToken))

	// json body 参数
	body := url.Values{}
	// 拼接出最终的百度存储地址
	baiduPath := strings.Builder{}
	baiduPath.WriteString("/apps")
	if prefixPath != "" {
		baiduPath.WriteString("/")
		baiduPath.WriteString(prefixPath)
	}
	baiduPath.WriteString("/")
	baiduPath.WriteString(localFilePath)
	baiduFilePath = baiduPath.String()
	// 准备 body 参数
	body.Add("path", baiduPath.String())
	body.Add("size", strconv.FormatInt(fileSize, 10))
	body.Add("isdir", "0")
	blockListJson, _ := json.Marshal(blockList)
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

	for i := 0; i < 3; i++ {
		ret, err = utils.DoHttpRequest(ret, &client, req)
		if err != nil {
			continue
		}
		break
	}

	if ret.Errno != 0 {
		return
	}
	return
}
