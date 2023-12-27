package upload

import (
	"baidu_tool/utils"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
)

type SingleUploadReturn struct {
	Md5       string `json:"md5"`
	RequestId int    `json:"request_id"`
}

func SingleUpload(accessToken string, uploadId string, baiduFilePath string, FileBytes []byte, partSeq int) (*SingleUploadReturn, error) {
	ret := new(SingleUploadReturn)

	payload := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(payload)
	fileWriter, err := bodyWriter.CreateFormFile("file", "file")
	if err != nil {
		return ret, err
	}
	_, err = io.Copy(fileWriter, bytes.NewReader(FileBytes))
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
	params.Set("path", baiduFilePath)
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

	for i := 0; i < 5; i++ {
		ret, err = utils.DoHttpRequest(ret, &client, req)
		if err != nil {
			continue
		}
		break
	}

	if ret.Md5 == "" {
		log.Printf("ret: %v; err: %v", ret, err)
		return ret, errors.New("md5 is empty")
	}
	return ret, nil
}
