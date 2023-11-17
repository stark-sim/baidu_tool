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

type CreateParam struct {
	Path      string   `json:"path"`
	Size      string   `json:"size"`
	IsDir     string   `json:"isdir"`
	BlockList []string `json:"block_list"`
	UploadId  string   `json:"uploadid"`
	RType     int      `json:"rtype"`
}

type CreateReturn struct {
	Errno int    `json:"errno"`
	Path  string `json:"path"`
}

func Create(accessToken string, path string, size int64, blockList []string, UploadId string) (*CreateReturn, error) {
	ret := &CreateReturn{}

	protocal := "https"
	host := "pan.baidu.com"
	router := "/rest/2.0/xpan/file?method=create&access_token=%s"
	uri := protocal + "://" + host + router
	realUrl, _ := url.Parse(fmt.Sprintf(uri, accessToken))

	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("User-Agent", "pan.baidu.com")

	body := url.Values{}
	body.Add("path", "/apps/"+path)
	body.Add("size", strconv.FormatInt(size/8, 10))
	body.Add("isdir", "0")
	bts, _ := json.Marshal(blockList)
	body.Add("block_list", string(bts))
	body.Add("uploadid", UploadId)
	body.Add("rtype", "2")

	req := http.Request{
		Method: "POST",
		URL:    realUrl,
		Header: header,
		Body:   io.NopCloser(strings.NewReader(body.Encode())),
	}

	ret, err := utils.DoHttpRequest(ret, &http.Client{}, &req)
	if err != nil {
		return ret, err
	}

	if ret.Errno != 0 {
		return ret, errors.New("call create failed")
	}
	return ret, nil
}
