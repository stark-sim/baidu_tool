package upload

import (
	"baidu_tool/uploadsdk/utils"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// Precreate
//
// RETURNS:
//   - PrecreateReturn: precreate return
//   - error: the return error if any occurs
func Precreate(accessToken string, arg *PrecreateArg) (PrecreateReturn, error) {
	ret := PrecreateReturn{}

	protocal := "https"
	host := "pan.baidu.com"
	router := "/rest/2.0/xpan/file?method=precreate&"
	uri := protocal + "://" + host + router

	params := url.Values{}
	params.Set("access_token", accessToken)
	uri += params.Encode()

	headers := map[string]string{
		"Host":         host,
		"Content-Type": "application/x-www-form-urlencoded",
	}

	postBody := url.Values{}
	postBody.Add("path", arg.Path)
	postBody.Add("size", strconv.FormatUint(arg.Size, 10))
	blockListJson, _ := json.Marshal(arg.BlockList)
	postBody.Add("block_list", string(blockListJson))
	postBody.Add("isdir", "0")
	postBody.Add("autoinit", "1")
	// 当path冲突且block_list不同时，进行重命名
	postBody.Add("rtype", "2")

	body, _, err := utils.DoHTTPRequest(uri, strings.NewReader(postBody.Encode()), headers)
	if err != nil {
		return ret, err
	}
	if err = json.Unmarshal([]byte(body), &ret); err != nil {
		return ret, errors.New("unmarshal precreate body failed,body")
	}
	if ret.Errno != 0 {
		return ret, errors.New("call precreate failed")
	}
	return ret, nil
}
