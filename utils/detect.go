package utils

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/oschwald/geoip2-golang"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/json"
	M "github.com/sagernet/sing/common/metadata"
)

func RetureIp(out adapter.Outbound) (string, error) {
	conn, err := out.DialContext(context.Background(), "tcp", M.ParseSocksaddrHostPort("clients3.google.com", 80))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	ip, err := RetureIpConn(conn)
	// log.Println(out.Tag(), ip)
	if err != nil {
		// log.Println(err)
		urls, errs := GetRemoteIps(out)
		if errs != nil {
			return "", errs
		}
		return urls, nil
	}
	return ip, nil
}

func RetureIpConn(conn net.Conn) (string, error) {
	return conn.RemoteAddr().String(), nil
}

func GetCountry(ip string, db *geoip2.Reader) (string, error) {
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}
	// If you are using strings that may be invalid, check that ip is not nil
	record, err := db.Country(net.ParseIP(ip))
	if err != nil {
		return "", err
	}
	return record.Country.IsoCode, nil
}

func Decect(out adapter.Outbound, conn net.Conn, db *geoip2.Reader) (string, string, error) {
	ip, err := RetureIpConn(conn)
	if err != nil {
		return "", "", err
	}
	if ip == ":0" {
		ips, err := GetRemoteIps(out)
		if err != nil {
			return "", "", err
		}
		ip, country := parseIP(ips)
		// log.Println(out.Tag(), ip, country)
		return ip, country, nil
	}
	country, err := GetCountry(ip, db)
	if err != nil {
		return "", "", err
	}
	return ip, country, nil
}

func CheckGeoLite2(filename string) error {

	// File does not exist, download it
	resp, err := http.Get("https://ghp.ci/https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-Country.mmdb")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

var httpClient *http.Client

func GetRemoteIp(out adapter.Outbound) ([]byte, error) {
	urls := "http://api-ipv4.ip.sb/ip"
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return out.DialContext(ctx, "tcp", M.ParseSocksaddr(addr))
		},
		ForceAttemptHTTP2: true,
	}
	httpClient = &http.Client{
		Transport: transport,
	}
	defer httpClient.CloseIdleConnections()
	parseURL, err := url.Parse(urls)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", parseURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "curl/7.88.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// _, err = bufio.Copy(os.Stdout, resp.Body)
	// jsonMarp, err := json.MarshalIndent(resp.Body, "", "\t")
	jsonMarp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return jsonMarp, nil
}

func GetRemoteIps(out adapter.Outbound) (string, error) {
	urls := "http://ipwho.is"
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return out.DialContext(ctx, "tcp", M.ParseSocksaddr(addr))
		},
		ForceAttemptHTTP2: true,
	}
	httpClient = &http.Client{
		Transport: transport,
	}
	defer httpClient.CloseIdleConnections()
	parseURL, err := url.Parse(urls)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", parseURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "curl/7.88.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// _, err = bufio.Copy(os.Stdout, resp.Body)
	// jsonMarp, err := json.MarshalIndent(resp.Body, "", "\t")
	jsonMarp, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(jsonMarp), nil
}

// convert string to json format
func parseIP(ip string) (string, string) {
	var result map[string]interface{}
	json.Unmarshal([]byte(ip), &result)
	// log.Println(result)
	if ip == "" {
		return "", ""
	}
	var ips, country string
	ips, exists := result["ip"].(string)
	if !exists {
		ips = ""
	}
	country, existss := result["country_code"].(string)
	if !existss {
		var ess bool
		country, ess = result["country"].(string)
		if !ess {
			country = ""
		}
	}
	return ips, country
}

func GetIP(out adapter.Outbound) (string, string, error) {
	ip, err := GetRemoteIps(out)
	if err != nil {
		return "", "", err
	}
	ip, country := parseIP(ip)
	return ip, country, nil
}
