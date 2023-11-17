package upload

import (
	"baidu_tool/utils"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type UploadReturn struct {
	Md5       string `json:"md5"`
	RequestId int    `json:"request_id"`
}

func Upload(accessToken string, uploadId string, localFilePath string, partSeq int) (*UploadReturn, error) {
	ret := &UploadReturn{}

	//打开文件句柄操作
	fileHandle, err := os.Open(localFilePath)
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
	_, err = fileHandle.Read(buf)
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
	_, err = io.Copy(fileWriter, bytes.NewReader(buf))
	if err != nil {
		return ret, err
	}
	if err = bodyWriter.Close(); err != nil {
		return nil, err
	}
	contentType := bodyWriter.FormDataContentType()

	uri := "https://d.pcs.baidu.com/rest/2.0/pcs/superfile2?method=upload&"

	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("path", "/apps/"+localFilePath)
	params.Set("uploadid", uploadId)
	params.Set("partseq", strconv.Itoa(partSeq))
	uri += params.Encode()

	header := http.Header{}
	header.Set("Content-Type", contentType)
	header.Set("Host", "d.pcs.baidu.com")

	bts, err := io.ReadAll(payload)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(bts))

	req.Header = header

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConnsPerHost: -1,
		},
	}
	ret, err = utils.DoHttpRequest(ret, &client, req)
	if err != nil {
		return ret, err
	}

	if ret.Md5 == "" {
		return ret, errors.New("md5 is empty")

	}
	return ret, nil
}
