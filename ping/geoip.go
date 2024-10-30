package ping

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Dkwkoaca/singtools/utils"
	"github.com/oschwald/geoip2-golang"
	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
)

// GeoIPManager 地理位置管理器
type GeoIPManager struct {
	db        *geoip2.Reader
	dbPath    string
	cache     sync.Map
	resources *ResourcePool
	config    *Config
}

// NewGeoIPManager 创建地理位置管理器
func NewGeoIPManager(config *Config) (*GeoIPManager, error) {
	if !utils.FileExists(config.GeoIPDBPath) {
		Download(*config)
	}
	db, err := geoip2.Open(config.GeoIPDBPath)
	if err != nil {
		return nil, fmt.Errorf("open GeoIP database failed: %w", err)
	}

	return &GeoIPManager{
		db:     db,
		dbPath: config.GeoIPDBPath,
		config: config,
	}, nil
}

// GetCountry 获取国家信息
func (g *GeoIPManager) GetCountry(ip string) (string, error) {
	// 检查缓存
	if country, ok := g.cache.Load(ip); ok {
		return country.(string), nil
	}

	// 清理IP地址格式
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	record, err := g.db.Country(parsedIP)
	if err != nil {
		return "", err
	}

	// 更新缓存
	g.cache.Store(ip, record.Country.IsoCode)
	return record.Country.IsoCode, nil
}

// IPInfo IP信息结构
type IPInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	City    string `json:"city,omitempty"`
}

// GetRemoteIPInfo 获取远程IP信息
func (g *GeoIPManager) GetRemoteIPInfo(ctx context.Context, out adapter.Outbound) (*IPInfo, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return out.DialContext(ctx, network, M.ParseSocksaddr(addr))
			},
		},
		Timeout: g.config.Timeout,
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://ip-api.com/json/", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info IPInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Close 关闭数据库
func (g *GeoIPManager) Close() error {
	// remove db file if exists
	os.Remove(g.config.GeoIPDBPath)
	return g.db.Close()
}

func Download(c Config) error {
	if utils.FileExists(c.GeoIPDBPath) {
		return nil
	}
	fmt.Println("Download")
	url := "https://ghp.ci/https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-Country.mmdb"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(c.GeoIPDBPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
