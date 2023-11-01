package baidu_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DirResp 接口文件夹返回
type DirResp struct {
	Errno     int          `json:"errno"`
	Cursor    int          `json:"cursor"`
	List      []*FileOrDir `json:"list"`
	RequestID string       `json:"request_id"`
	HasMore   int8         `json:"has_more"`
	Errmsg    string       `json:"errmsg"`
}

// FileOrDir 文件或文件夹结构
type FileOrDir struct {
	IsDir          int8   `json:"is_dir"`
	Size           int64  `json:"size"`
	Path           string `json:"path"`
	ServerFilename string `json:"server_filename"`
	FsId           int64  `json:"fs_id"`
	MD5            string `json:"md5"`
}

// GetFileOrDirResp 获取到路径所指的文件或文件夹的接口返回
func GetFileOrDirResp(accessToken string, filePath string) (*DirResp, error) {
	preUrl := "http://pan.baidu.com/rest/2.0/xpan/multimedia?method=listall&access_token=%s&path=%s&recursion=1"
	_url := fmt.Sprintf(preUrl, accessToken, url.PathEscape(filePath))
	resp, err := http.Get(_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var dirResp DirResp
	if err = json.Unmarshal(respBytes, &dirResp); err != nil {
		return nil, err
	}
	if dirResp.Errno != 0 {
		return nil, fmt.Errorf("dir resp not right")
	}
	return &dirResp, nil
}

// DivideDirAndFile 分离文件路径中的 / 来找出文件夹和文件名
func DivideDirAndFile(filePath string) (dir string, file string, err error) {
	lastIndex := strings.LastIndex(filePath, "/")
	if lastIndex == -1 {
		return "", "", fmt.Errorf("not found /")
	}
	return filePath[:lastIndex+1], filePath[lastIndex+1:], nil
}
