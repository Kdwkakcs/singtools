package cache

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
)

// compressContent 压缩内容
func compressContent(content []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(content)
	if err != nil {
		return nil, err
	}
	err = zw.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decompressContent 解压缩内容
func decompressContent(compressedContent []byte) ([]byte, error) {
	buf := bytes.NewBuffer(compressedContent)
	zr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	content, err := ioutil.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	return content, nil
}

// hashContent 计算内容的哈希值
func hashContent(content []byte) [32]byte {
	return sha256.Sum256(content)
}

// fetchContent 从 URL 获取内容，添加超时处理
func fetchContent(url string, client *http.Client) ([]byte, error) {
	// Use the passed client with a timeout
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for URL %s: %v", url, err)
	}

	return body, nil
}
