package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func DoHttpRequest[T any](respVar *T, client *http.Client, req *http.Request) (*T, error) {
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("client do err: %v\n", err)
		return respVar, err
	}
	defer resp.Body.Close()
	respBts, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("io readAll body err: %v\n", err)
		return respVar, err
	}
	if err = json.Unmarshal(respBts, respVar); err != nil {
		fmt.Printf("json unmarshal err: %v\n", err)
		return respVar, err
	}
	return respVar, nil
}
