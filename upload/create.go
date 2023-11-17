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
	"strconv"
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

	jsonBody, err := json.Marshal(CreateParam{
		Path:      path,
		Size:      strconv.FormatInt(size, 10),
		IsDir:     "0",
		BlockList: blockList,
		UploadId:  UploadId,
		RType:     2,
	})

	req := http.Request{
		Method: "POST",
		URL:    realUrl,
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(jsonBody)),
	}

	ret, err = utils.DoHttpRequest(ret, &http.Client{}, &req)
	if err != nil {
		return ret, err
	}

	if ret.Errno != 0 {
		return ret, errors.New("call create failed")
	}
	return ret, nil
}
