package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// FileExists 检查文件是否存在
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// DownloadGeoLite2 下载 GeoIP 数据库
func DownloadGeoLite2(filepath string) error {
	// GeoLite2 数据库下载地址
	url := "https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/download/GeoLite2-Country.mmdb"

	// 创建文件
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer out.Close()

	// 发起 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// 写入文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}
