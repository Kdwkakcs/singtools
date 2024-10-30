package cache

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Dkwkoaca/singtools/get"
	"github.com/sagernet/sing/common/json"
	bolt "go.etcd.io/bbolt"
)

type CacheManager struct {
	db   *bolt.DB
	path string
	mu   sync.Mutex
}

// 添加常量定义
const (
	bucketURLData        = "URLData"
	bucketURLs           = "URLs"
	bucketUpdatedContent = "UpdatedContent"
)

// URLData 包含 URL 的内容、哈希值和更新时间
type URLData struct {
	URL     string
	Content []byte // 压缩后的内容
	Hash    [32]byte
	Updated time.Time
}

// NewCacheManager 打开数据库并返回 CacheManager 实例
func NewCacheManager(dbPath string) (*CacheManager, error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	// 创建所需的 bucket
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketURLData))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(bucketURLs))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &CacheManager{db: db, path: dbPath}, nil
}

// Close 关闭数据库
func (manager *CacheManager) Close() error {
	return manager.db.Close()
}

func (manager *CacheManager) Stats() bolt.Stats {
	return manager.db.Stats()
}

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

// StoreURLData 存储最新的 URL 数据，只保留最新的版本
func (manager *CacheManager) StoreURLData(data URLData) error {
	compressedContent, err := compressContent(data.Content)
	if err != nil {
		return err
	}
	data.Content = compressedContent

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(data)
	if err != nil {
		return err
	}

	// 更新数据库
	return manager.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketURLData))
		if err != nil {
			return err
		}

		// 直接使用 URL 作为唯一键，覆盖以前的数据
		return bucket.Put([]byte(data.URL), buf.Bytes())
	})
}

// GetURLData 获取最新的 URL 数据
func (manager *CacheManager) GetURLData(url string) (URLData, error) {
	if url == "" {
		return URLData{}, fmt.Errorf("empty URL provided")
	}

	var data URLData
	err := manager.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketURLData))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", bucketURLData)
		}

		v := bucket.Get([]byte(url))
		if v == nil {
			return fmt.Errorf("no data found for URL: %s", url)
		}

		return gob.NewDecoder(bytes.NewReader(v)).Decode(&data)
	})
	if err != nil {
		return URLData{}, fmt.Errorf("failed to get URL data: %w", err)
	}

	content, err := decompressContent(data.Content)
	if err != nil {
		return URLData{}, fmt.Errorf("failed to decompress content: %w", err)
	}
	data.Content = content

	return data, nil
}

func (manager *CacheManager) GetAllURLData() (map[string]URLData, error) {
	urlData := make(map[string]URLData)
	err := manager.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketURLData))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			var data URLData
			buf := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&data)
			if err != nil {
				return err
			}
			content, err := decompressContent(data.Content)
			if err != nil {
				return err
			}
			data.Content = content
			urlData[string(k)] = data
			return nil
		})
	})
	return urlData, err
}

func (manager *CacheManager) GetSubURLsData(urls []string) (map[string]URLData, error) {
	urlData := make(map[string]URLData)
	err := manager.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketURLData))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		for _, url := range urls {
			v := bucket.Get([]byte(url))
			if v == nil {
				return fmt.Errorf("URL data not found")
			}
			var data URLData
			buf := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&data)
			if err != nil {
				return err
			}
			content, err := decompressContent(data.Content)
			if err != nil {
				return err
			}
			data.Content = content
			urlData[url] = data
		}
		return nil
	})
	return urlData, err
}

func (manager *CacheManager) getUpdateContent(urls []string) string {
	var content []interface{}
	urlData, err := manager.GetSubURLsData(urls)
	if err != nil {
		log.Println("Error getting URLs:", err)
		return ""
	}
	for _, v := range urlData {
		var data []interface{}
		err := json.Unmarshal(v.Content, &data)
		if err != nil {
			continue
		}
		content = append(content, data...)
	}
	// remove duplicates
	unique := get.Unique(content)
	unique = get.RemoveDuplicateOutbounds(unique)
	jsonStr, _ := json.Marshal(unique)
	return string(jsonStr)
}

func (manager *CacheManager) GetAllProxiesFromDB() (string, error) {
	urlData, err := manager.GetAllURLData()
	if err != nil {
		return "", err
	}
	proxies := make([]interface{}, 0, len(urlData)*1000)
	for _, v := range urlData {
		var data []interface{}
		err := json.Unmarshal(v.Content, &data)
		if err != nil {
			continue
		}
		proxies = append(proxies, data...)
	}
	unique := get.Unique(proxies)
	unique = get.RemoveDuplicateOutbounds(unique)
	jsonStr, _ := json.Marshal(unique)
	return string(jsonStr), nil
}

// AddURL 添加 URL 到数据库
func (manager *CacheManager) AddURL(url string) error {
	return manager.db.Update(func(tx *bolt.Tx) error {
		urlsBucket, err := tx.CreateBucketIfNotExists([]byte(bucketURLs))
		if err != nil {
			return err
		}
		return urlsBucket.Put([]byte(url), []byte{})
	})
}

// GetAllURLs 获取所有存储的 URL
func (manager *CacheManager) GetAllURLs() ([]string, error) {
	var urls []string
	err := manager.db.View(func(tx *bolt.Tx) error {
		urlsBucket := tx.Bucket([]byte(bucketURLs))
		if urlsBucket == nil {
			return fmt.Errorf("URLs bucket not found")
		}
		return urlsBucket.ForEach(func(k, v []byte) error {
			urls = append(urls, string(k))
			return nil
		})
	})
	return urls, err
}

func (manager *CacheManager) GetURLs() ([]string, error) {
	var urls []string
	err := manager.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketURLData))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		return bucket.ForEach(func(k, v []byte) error {
			urls = append(urls, string(k))
			return nil
		})
	})
	return urls, err
}

// CheckAndUpdateURL 检查并更新 URL 的内容
// CheckAndUpdateURL checks and updates the content of a URL
// Logs the successfully updated URLs and creates a new table for new updates
func (manager *CacheManager) CheckAndUpdateURL(url string, client *http.Client) (bool, error) {
	content, err := fetchContent(url, client)
	if err != nil {
		return false, err
	}

	hash := hashContent(content)
	data, err := get.ParseContent(string(content))
	if err != nil {
		return false, fmt.Errorf("failed to parse content for URL %s: %v", url, err)
	}
	contentStr, err := json.Marshal(data)
	if err != nil {
		return false, fmt.Errorf("failed to marshal content for URL %s: %v", url, err)
	}

	existingData, err := manager.GetURLData(url)
	if err != nil && err.Error() != "URL data not found" {
		return false, err
	}

	if err != nil || existingData.Hash != hash {
		updatedData := URLData{
			URL:     url,
			Content: contentStr,
			Hash:    hash,
			Updated: time.Now(),
		}

		err = manager.StoreURLData(updatedData)
		if err != nil {
			return false, fmt.Errorf("failed to store updated data for URL %s: %v", url, err)
		}

		// Store updated URL in URLs bucket
		err = manager.AddURL(url)
		if err != nil {
			log.Printf("Failed to log URL %s to URLs bucket: %v\n", url, err)
		}

		// Log new updates in a new bucket (UpdatedContent)
		err = manager.db.Update(func(tx *bolt.Tx) error {
			updatedBucket, err := tx.CreateBucketIfNotExists([]byte(bucketUpdatedContent))
			if err != nil {
				return err
			}
			return updatedBucket.Put([]byte(url), []byte(contentStr))
		})
		if err != nil {
			log.Printf("Failed to log updated content for URL %s: %v\n", url, err)
		}

		fmt.Printf("URL %s has been updated.\n", url)
		return true, nil
	} else {
		fmt.Printf("URL %s has no changes.\n", url)
		return false, nil
	}
}

// LoadExistingData 读取已有数据库内容
func (manager *CacheManager) LoadExistingData() error {
	return manager.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketURLData))
		if bucket == nil {
			return fmt.Errorf("URLData bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var data URLData
			buf := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buf)
			err := dec.Decode(&data)
			if err != nil {
				return err
			}

			// 输出读取的内容
			fmt.Printf("URL: %s, Last Updated: %s\n", data.URL, data.Updated)
			return nil
		})
	})
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

// hashContent 计算内容的哈希值
func hashContent(content []byte) [32]byte {
	return sha256.Sum256(content)
}

// UpdateURLs 更新多个 URL 的数据
// UpdateURLs updates the data for multiple URLs and logs them
func (manager *CacheManager) UpdateURLs(urls []string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided")
	}

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		updatedURLs []string
		client      = &http.Client{Timeout: 10 * time.Second}
	)

	// 使用带缓冲的通道限制并发数
	semaphore := make(chan struct{}, 10)
	errChan := make(chan error, len(urls))

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			updated, err := manager.CheckAndUpdateURL(u, client)
			if err != nil {
				errChan <- fmt.Errorf("error updating %s: %w", u, err)
				return
			}

			if updated {
				mu.Lock()
				updatedURLs = append(updatedURLs, u)
				mu.Unlock()
			}
		}(url)
	}

	// 等待所有更新完成
	wg.Wait()
	close(errChan)

	// 处理错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred: %v", errs)
	}

	// 处理更新的内容
	if len(updatedURLs) > 0 {
		content := manager.getUpdateContent(updatedURLs)
		err := manager.storeUpdatedContent(content)
		if err != nil {
			return fmt.Errorf("failed to store updated content: %w", err)
		}
	}

	return nil
}

// 新增辅助方法，存储更新的内容
func (manager *CacheManager) storeUpdatedContent(content string) error {
	return manager.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketUpdatedContent))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		compressedContent, err := compressContent([]byte(content))
		if err != nil {
			return fmt.Errorf("failed to compress content: %w", err)
		}

		return bucket.Put([]byte("merged"), compressedContent)
	})
}

func (manager *CacheManager) GetUpdated() string {
	var content []byte
	err := manager.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketUpdatedContent))
		if bucket == nil {
			return fmt.Errorf("UpdatedContent bucket not found")
		}

		mergedContent := bucket.Get([]byte("merged"))
		if mergedContent == nil {
			return fmt.Errorf("merged content not found")
		}

		// 解压缩内容
		decompressed, err := decompressContent(mergedContent)
		if err != nil {
			return fmt.Errorf("failed to decompress content: %v", err)
		}
		content = decompressed
		return nil
	})

	if err != nil {
		log.Printf("Failed to get updated content: %v\n", err)
		return ""
	}

	return string(content)
}

func (manager *CacheManager) mergeAllURLs() (string, error) {
	urlData, err := manager.GetAllURLData()
	if err != nil {
		log.Println("Error getting URLs:", err)
		return "", err
	}

	var all_node []interface{}
	for _, url := range urlData {
		// urls := url.URL
		// context := url.Content
		var jsondata []interface{}
		err := json.Unmarshal(url.Content, &jsondata)
		if err != nil {
			// log.Print(urls, " ", string(context))
			continue
		}
		all_node = append(all_node, jsondata...)
	}
	unique := get.Unique(all_node)
	unique = get.RemoveDuplicateOutbounds(unique)
	fmt.Println("all nodes: ", len(all_node), ", unique nodes: ", len(unique))

	uniqueStr, err := json.Marshal(unique)
	if err != nil {
		return "", err
	}
	return string(uniqueStr), nil
}

func (manager *CacheManager) Glimpse() {
	data, err := manager.GetAllURLData()
	if err != nil {
		log.Println("Error getting URLs:", err)
		return
	}
	var all_node []interface{}
	for _, url := range data {
		urls := url.URL
		context := url.Content

		var jsondata []interface{}
		err := json.Unmarshal(url.Content, &jsondata)
		if err != nil {
			log.Print(urls, " ", string(context))
		}

		lens := len(context)
		if lens > 200 {
			context = context[0:200]
		}

		all_node = append(all_node, jsondata...)
		fmt.Println("The url: ", urls, ", content: ", string(context), ", with length: ", lens, ", with nodes: ", len(jsondata))
	}
	unique := get.Unique(all_node)
	fmt.Println("all nodes: ", len(all_node), ", unique nodes: ", len(unique))
}
