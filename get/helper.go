package get

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/levigross/grequests"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

var (
	defaultRequestOptions = &grequests.RequestOptions{
		RequestTimeout: time.Second * 10,
		DialTimeout:    time.Second * 10,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (compatible; YourAppName/1.0)",
		},
	}
	skipTypes = []string{"selector", "block", "dns", "urltest", "tor", "direct"}
)

// GetLinksFromHTML 解析HTML内容并提取所有链接
func GetLinksFromHTML(url string) ([]string, error) {
	res, err := grequests.Get(url, defaultRequestOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL %s: %v", url, err)
	}

	doc, err := html.Parse(strings.NewReader(res.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %v", err)
	}

	var links []string
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attr := range node.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") && !strings.Contains(attr.Val, "t.me") {
					links = append(links, attr.Val)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)
	return links, nil
}

// IsHTMLContent 检查响应是否为HTML内容
func IsHTMLContent(resp *grequests.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "text/html")
}

// ExtractLinks 从YAML配置中提取所有链接
func ExtractLinks(ymlContent string) ([]string, error) {
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(ymlContent), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	var links []string
	var wg sync.WaitGroup
	var mu sync.Mutex
	concurrency := 10
	semaphore := make(chan struct{}, concurrency)

	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	excludeFileRegex := regexp.MustCompile(`(?i)\.(apk|exe|dmg|pkg|msi|deb|rpm)$`)

	for key, value := range data {
		items, ok := value.([]interface{})
		if !ok {
			continue
		}
		for _, v := range items {
			urlStr, ok := v.(string)
			if !ok {
				continue
			}

			switch key {
			case "category":
				wg.Add(1)
				semaphore <- struct{}{}
				go func(url string) {
					defer func() {
						<-semaphore
						wg.Done()
					}()
					res, err := GetLinksFromHTML(url)
					if err != nil {
						log.Printf("Error fetching links from %s: %v", url, err)
						return
					}
					mu.Lock()
					links = append(links, res...)
					mu.Unlock()
				}(urlStr)

			case "url":
				wg.Add(1)
				semaphore <- struct{}{}
				go func(url string) {
					defer func() {
						<-semaphore
						wg.Done()
					}()
					resp, err := grequests.Get(url, defaultRequestOptions)
					if err != nil {
						log.Printf("Error fetching URL %s: %v", url, err)
						return
					}
					matchedURLs := urlRegex.FindAllString(resp.String(), -1)
					mu.Lock()
					links = append(links, matchedURLs...)
					mu.Unlock()
				}(urlStr)

			case "local":
				file, err := os.Open(urlStr)
				if err != nil {
					log.Printf("Error opening file %s: %v", urlStr, err)
					continue
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "http") && !excludeFileRegex.MatchString(line) {
						links = append(links, line)
					}
				}
			default:
				if !excludeFileRegex.MatchString(urlStr) {
					links = append(links, urlStr)
				}
			}
		}
	}

	wg.Wait()
	links = uniqueStrings(links)
	return links, nil
}

func Unique[T comparable](items []T) []T {
	seen := make(map[string]struct{}, len(items))
	result := make([]T, 0, len(items))

	for _, v := range items {
		key, err := json.Marshal(v)
		if err != nil {
			log.Fatal(err)
		}

		if _, ok := seen[string(key)]; !ok {
			seen[string(key)] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}

// uniqueStrings 去除字符串切片中的重复项
func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, item := range items {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// RemoveDuplicateOutbounds 去除重复的出站配置
func RemoveDuplicateOutbounds(outbounds []interface{}) []interface{} {
	seen := make(map[string]struct{})
	var result []interface{}
	for _, outbound := range outbounds {
		outboundMap, ok := outbound.(map[string]interface{})
		if !ok {
			continue
		}
		hash := generateOutboundHash(outboundMap)
		if _, exists := seen[hash]; !exists {
			seen[hash] = struct{}{}
			result = append(result, outbound)
		}
	}
	return result
}

// generateOutboundHash 生成出站配置的哈希值用于去重
func generateOutboundHash(outbound map[string]interface{}) string {
	// 排除tag字段
	outboundCopy := make(map[string]interface{})
	for k, v := range outbound {
		if k != "tag" {
			outboundCopy[k] = v
		}
	}
	jsonData, _ := json.Marshal(outboundCopy)
	hash := md5.Sum(jsonData)
	return hex.EncodeToString(hash[:])
}

// RenameOutbounds 重命名出站配置的tag字段
func RenameOutbounds(outbounds []interface{}) []interface{} {
	for i, outbound := range outbounds {
		outboundMap, ok := outbound.(map[string]interface{})
		if !ok {
			continue
		}
		outboundMap["tag"] = "Node" + strconv.Itoa(i+1)
	}
	return outbounds
}

// ContainsPrefix 检查字符串是否包含指定前缀
func ContainsPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// ExtractURLsFromContent 从内容中提取指定协议的链接并进行Base64编码
func ExtractURLsFromContent(content string) string {
	prefixes := []string{"vmess://", "vless://", "trojan://", "ss://", "ssr://", "hysteria2://", "hysteria://", "hy2://", "hy://", "tuic://"}
	var matchedLines []string
	for _, line := range strings.Split(content, "\n") {
		if ContainsPrefix(line, prefixes) {
			matchedLines = append(matchedLines, line)
		}
	}
	if len(matchedLines) == 0 {
		return ""
	}
	joined := strings.Join(matchedLines, "\n")
	return base64.StdEncoding.EncodeToString([]byte(joined))
}

// isBase64 检查字符串是否是有效的 Base64 编码
func isBase64(s string) bool {
	// Base64 编码字符串的长度必须是4的倍数
	if len(s)%4 != 0 {
		return false
	}

	// 使用 base64.RawStdEncoding 解码，忽略填充字符
	decoded, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return false
	}
	return len(decoded) > 0 || s == "" // 如果解码成功，返回 true
}
