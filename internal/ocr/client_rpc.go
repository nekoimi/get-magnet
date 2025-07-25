package ocr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

var (
	requestUrl = fmt.Sprintf("http://127.0.0.1:%d/ocr/file/json", Port)
	client     = &http.Client{
		Timeout: 10 * time.Second,
	}
)

type Response struct {
	Status int    `json:"status"`
	Result string `json:"result"`
}

func Call(imageBytes []byte) (string, error) {
	body, contentType, err := buildRequestBody(imageBytes)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", requestUrl, body)
	if err != nil {
		return "", err
	}

	// 设置 Content-Type
	req.Header.Set("Content-Type", contentType)

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("请求ocr服务异常: %s", err.Error())
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("请求ocr服务异常: status - %d", resp.StatusCode))
	}

	defer resp.Body.Close()

	result := new(Response)
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("读取ocr服务结果异常: %s", err.Error())
		return "", err
	}

	err = json.Unmarshal(bytes, result)
	if err != nil {
		log.Errorf("解析ocr服务结果异常: %s", err.Error())
		return "", err
	}

	if result.Status != 200 {
		return "", errors.New(fmt.Sprintf("ocr响应异常: %v", result))
	}

	log.Debugf("ocr服务响应：%v", result)

	return result.Result, nil
}

func buildRequestBody(imageBytes []byte) (*bytes.Buffer, string, error) {
	// 创建一个 buffer 和 multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 创建 form file 字段
	part, err := writer.CreateFormFile("image", "image.png")
	if err != nil {
		return nil, "", err
	}

	// 将文件内容复制到 form file 字段中
	_, err = io.Copy(part, bytes.NewBuffer(imageBytes))
	if err != nil {
		return nil, "", err
	}

	// 关闭 writer，写入结尾的 boundary
	err = writer.Close()
	if err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}
