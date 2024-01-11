package baidu_api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// DirRecursiveResp 接口文件夹返回
type DirRecursiveResp struct {
	Errno     int          `json:"errno"`
	Cursor    int          `json:"cursor"`
	List      []*FileOrDir `json:"list"`
	RequestID string       `json:"request_id"`
	HasMore   int8         `json:"has_more"`
	Errmsg    string       `json:"errmsg"`
}

// FileOrDir 文件或文件夹结构
type FileOrDir struct {
	IsDir          int8   `json:"isdir"`
	Size           int64  `json:"size"`
	Path           string `json:"path"`
	ServerFilename string `json:"server_filename"`
	FsId           int64  `json:"fs_id"`
	MD5            string `json:"md5"`
}

// GetFileOrDirResp 获取到路径所指的文件或文件夹的接口返回
func GetFileOrDirResp(accessToken string, filePath string) (*DirRecursiveResp, error) {
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
	var dirResp DirRecursiveResp
	if err = json.Unmarshal(respBytes, &dirResp); err != nil {
		return nil, err
	}
	if dirResp.Errno != 0 {
		return nil, fmt.Errorf("dir resp not right")
	}
	return &dirResp, nil
}

// DirListResp 接口文件夹列表返回
type DirListResp struct {
	Errno     int          `json:"errno"`
	List      []*FileOrDir `json:"list"`
	RequestID int64        `json:"request_id"`
	Guid      int          `json:"guid"`
}

// GetDirByList 使用列表方法，非递归，获取一个文件夹下的文件信息，递归最多 1000 个下级信息
func GetDirByList(accessToken string, dirPath string) (*DirListResp, error) {
	preUrl := "http://pan.baidu.com/rest/2.0/xpan/file?method=list&access_token=%s&dir=%s"
	_url := fmt.Sprintf(preUrl, accessToken, url.PathEscape(dirPath))
	resp, err := http.Get(_url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var dirResp DirListResp
	if err = json.Unmarshal(respBytes, &dirResp); err != nil {
		return nil, err
	}
	if dirResp.Errno != 0 {
		if dirResp.Errno == -9 {
			return nil, fmt.Errorf("NOT_FOUND")
		}
		return nil, fmt.Errorf("dir resp not right")
	}
	return &dirResp, nil
}
