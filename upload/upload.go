package upload

import (
	"baidu_tool/utils"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

type UploadArg struct {
	UploadId  string `json:"uploadid"`
	Path      string `json:"path"`
	LocalFile string `json:"local_file"`
	PartSeq   int    `json:"partseq"`
}

type UploadReturn struct {
	Md5       string `json:"md5"`
	RequestId int    `json:"request_id"`
}

func Upload(accessToken string, arg *UploadArg) (*UploadReturn, error) {
	ret := &UploadReturn{}

	//打开文件句柄操作
	fileHandle, err := os.Open(arg.LocalFile)
	if err != nil {
		return ret, errors.New("superfile open file failed")
	}
	defer fileHandle.Close()

	// 获取文件当前信息
	fileInfo, err := fileHandle.Stat()
	if err != nil {
		return ret, err
	}

	// 读取文件块
	buf := make([]byte, fileInfo.Size())
	n, err := fileHandle.Read(buf)
	if err != nil {
		if err != io.EOF {
			return ret, err
		}
	}

	payload := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(payload)
	fileWriter, err := bodyWriter.CreateFormFile("file", "file")
	if err != nil {
		return ret, err
	}
	_, err = io.Copy(fileWriter, bytes.NewReader(buf[0:n]))
	if err != nil {
		return ret, err
	}
	if err = bodyWriter.Close(); err != nil {
		return nil, err
	}
	contentType := bodyWriter.FormDataContentType()

	uri := "https://d.pcs.baidu.com/rest/2.0/pcs/superfile2?method=upload&access_token=%s&type=tmpfile&path=%suploadid=%s&partseq=%d"
	realUrl, _ := url.Parse(fmt.Sprintf(uri, accessToken, arg.Path, arg.UploadId, arg.PartSeq))

	header := http.Header{}
	header.Set("Content-Type", contentType)

	req := http.Request{
		Method: "POST",
		URL:    realUrl,
		Header: header,
		Body:   io.NopCloser(payload),
	}

	client := http.Client{}
	ret, err = utils.DoHttpRequest(ret, &client, &req)
	if err != nil {
		return ret, err
	}

	if ret.Md5 == "" {
		return ret, errors.New("md5 is empty")

	}
	return ret, nil
}
