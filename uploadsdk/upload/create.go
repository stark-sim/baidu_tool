package upload

import (
	"baidu_tool/uploadsdk/utils"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// create
//
// RETURNS:
//   - CreateReturn: create return
//   - error: the return error if any occurs
func Create(accessToken string, arg *CreateArg) (CreateReturn, error) {
	ret := CreateReturn{}

	protocal := "https"
	host := "pan.baidu.com"
	router := "/rest/2.0/xpan/file?method=create&"
	uri := protocal + "://" + host + router

	headers := map[string]string{
		"Host":         host,
		"Content-Type": "application/x-www-form-urlencoded",
	}

	params := url.Values{}
	params.Set("access_token", accessToken)
	uri += params.Encode()

	postBody := url.Values{}
	postBody.Add("rtype", "2")
	postBody.Add("path", arg.Path)
	postBody.Add("size", strconv.FormatUint(arg.Size, 10))
	postBody.Add("isdir", "0")
	js, _ := json.Marshal(arg.BlockList)
	postBody.Add("block_list", string(js))
	postBody.Add("uploadid", arg.UploadId)

	var body string
	var err error

	body, _, err = utils.DoHTTPRequest(uri, strings.NewReader(postBody.Encode()), headers)
	if err != nil {
		return ret, err
	}
	if err = json.Unmarshal([]byte(body), &ret); err != nil {
		return ret, errors.New("unmarshal create body failed")
	}
	if ret.Errno != 0 {
		return ret, errors.New("call create failed")
	}
	return ret, nil

}
