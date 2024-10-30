package cn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/Dkwkoaca/singtools/get"
	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// add function to download the db file from the internet
func downloadDB(filename string) error {
	resp, err := http.Get("https://github.com/P3TERX/GeoLite.mmdb/releases/latest/download/GeoLite2-Country.mmdb")
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

func removeDB(filename string) error {
	if err := os.Remove(filename); err != nil {
		return err
	}
	return nil
}

// lookupIP resolves a domain name to its first IP address.
func lookupIP(domain string) (string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IPs found for domain %s", domain)
	}
	return ips[0].String(), nil
}

// queryCountryByIP finds the country for a given IP address using a GeoIP database.
func queryCountryByIP(ip string, db *geoip2.Reader) (string, error) {
	ipAddress := net.ParseIP(ip)
	if ipAddress == nil {
		return "Invalid IP format", nil
	}

	record, err := db.Country(ipAddress)
	if err != nil {
		return "", err
	}

	return record.Country.IsoCode, nil
}

// 从URL下载YAML内容
func downloadYAML(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func convertToYaml(data []interface{}) []interface{} {
	var res []interface{}
	var outbound []interface{}
	for _, d := range data {
		// convert to yaml
		str, err := yaml.Marshal(&d)
		if err != nil {
			continue
		}
		proxy := get.NodeToSingbox(string(str), "clash")
		err = json.Unmarshal([]byte(proxy), &outbound)
		if err != nil {
			continue
		}
		if outbound[0] != nil {
			res = append(res, outbound[0])
		} else {
			fmt.Println(string(str))
		}
	}
	return res
}

// main is the entry point for the program.
func CNNodeTest(inputFile string, urlPath string, dbPath string, outputFile string, outputJson string) {
	if (inputFile == "" && urlPath == "") || (inputFile != "" && urlPath != "") {
		log.Fatal("Both input and url file paths are required")
	}
	if outputFile == "" || outputJson == "" {
		log.Fatal("Output file paths are required")
	}
	var fileContents string
	var err error
	if inputFile != "" {
		log.Println("Reading file:", inputFile)
		fileContent, err := ioutil.ReadFile(inputFile)
		if err != nil {
			log.Fatal("Error reading file:", err)
		}
		fileContents = string(fileContent)
	} else if urlPath != "" {
		log.Println("Downloading file:", urlPath)
		fileContents, err = get.GetUrl(string(urlPath))
		if err != nil {
			log.Fatal("Error downloading YAML file:", err)
		}
	}

	// parse the yaml using viper
	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer([]byte(fileContents)))
	if err != nil {
		log.Fatal("Error reading YAML file:", err)
	}

	// get the proxies from the yaml
	var proxies []interface{}
	if err := viper.UnmarshalKey("proxies", &proxies); err != nil {
		log.Fatal("Error unmarshalling proxies:", err)
	}

	outs := convertToYaml(proxies)

	// outs, err := mergeOutbounds(paths, prefixs, int64(*concurrencyLimiterFlag))
	fmt.Println(len(outs))
	outBytes, err := json.MarshalIndent(outs, "", "  ")
	if err != nil {
		log.Fatal("Error marshalling filtered servers to JSON:", err)
	}

	if err := ioutil.WriteFile(outputJson, outBytes, 0644); err != nil {
		log.Fatal("Error writing filtered servers to file:", err)
	}

	// // 下载 GeoIP 数据库
	if dbPath == "" {
		dbPath = "GeoLite2-Country.mmdb"
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if err := downloadDB(dbPath); err != nil {
			log.Fatal("Error downloading GeoIP database:", err)
		}
	}
	db, err := geoip2.Open(dbPath)
	if err != nil {
		log.Fatal("Error opening GeoIP database:", err)
	}
	defer db.Close()

	var filteredServers []interface{}

	var mu sync.Mutex     // 用于保护 filteredServers 的互斥锁
	var wg sync.WaitGroup // 用于等待所有协程完成
	// 限制并发数量的通道，大小为20
	concurrencyLimit := make(chan struct{}, 5000)

	for _, node := range outs {
		server := node // 创建当前循环变量的本地副本
		if server == nil {
			fmt.Println("Error parsing JSON: empty server object", server)
			continue
		}
		concurrencyLimit <- struct{}{} // 获取执行权限
		wg.Add(1)

		go func() {
			defer wg.Done()                       // 确保在函数退出时通知 WaitGroup
			defer func() { <-concurrencyLimit }() // 释放执行权限

			servers, ok := server.(map[string]interface{})
			if !ok {
				log.Fatal("Error parsing JSON: outbound entry is not an object", server)
				return
			}
			if len(servers) == 0 {
				return
			}
			ips, exists := servers["server"]
			if !exists {
				log.Println("Error parsing JSON: server key not found", servers["tag"], servers, node)
				return
			}
			ipStr, ok := ips.(string)
			if !ok {
				log.Fatal("Error parsing server IP: value is not a string")
				return
			}
			ip, err := lookupIP(ipStr)
			if err != nil {
				// 错误处理
				return
			}
			country, err := queryCountryByIP(ip, db)
			if err != nil {
				// 错误处理
				return
			}
			if country == "CN" {
				mu.Lock() // 在添加到 filteredServers 前加锁
				filteredServers = append(filteredServers, servers)
				mu.Unlock() // 解锁
			}
		}()
	}
	wg.Wait() // 等待所有协程完成

	config := map[string]interface{}{
		"log":       map[string]interface{}{},
		"dns":       map[string]interface{}{},
		"ntp":       map[string]interface{}{},
		"inbounds":  []interface{}{},
		"outbounds": []interface{}{},
		"route":     map[string]interface{}{},
	}
	fmt.Printf("Found %d servers in CN\n", len(filteredServers))
	if len(filteredServers) > 0 {
		outputData := config
		outputData["outbounds"] = filteredServers
		outputBytes, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			log.Fatal("Error marshalling filtered servers to JSON:", err)
		}

		if err := ioutil.WriteFile(outputFile, outputBytes, 0644); err != nil {
			log.Fatal("Error writing filtered servers to file:", err)
		}
		fmt.Printf("Filtered servers saved to %s\n", outputFile)
	} else {
		fmt.Println("No servers located in CN were found.")
	}

	// remove downloaded Geolite
	if err := os.Remove(dbPath); err != nil {
		log.Fatal("Error removing GeoLite database:", err)
	}
}
