package upload

import (
	"baidu_tool/uploadsdk/utils"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"strconv"
)

func Upload(accessToken string, arg *UploadArg) (UploadReturn, error) {
	ret := UploadReturn{}

	//打开文件句柄操作
	fileHandle, err := os.Open(arg.LocalFile)
	if err != nil {
		return ret, errors.New("superfile2 open file failed")
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

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("file", "file")
	if err != nil {
		return ret, err
	}

	//iocopy
	_, err = io.Copy(fileWriter, bytes.NewReader(buf[0:n]))
	if err != nil {
		return ret, err
	}

	bodyWriter.Close()

	protocal := "https"
	host := "d.pcs.baidu.com"
	router := "/rest/2.0/pcs/superfile2?method=upload&"
	uri := protocal + "://" + host + router

	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("path", arg.Path)
	params.Set("uploadid", arg.UploadId)
	params.Set("partseq", strconv.Itoa(arg.Partseq))
	uri += params.Encode()

	contentType := bodyWriter.FormDataContentType()
	headers := map[string]string{
		"Host":         host,
		"Content-Type": contentType,
	}

	body, _, err := utils.SendHTTPRequest(uri, bodyBuf, headers)
	if err != nil {
		return ret, err
	}
	if err := json.Unmarshal([]byte(body), &ret); err != nil {
		return ret, errors.New("unmarshal body failed")

	}
	if ret.Md5 == "" {
		return ret, errors.New("md5 is empty")

	}
	return ret, nil
}
