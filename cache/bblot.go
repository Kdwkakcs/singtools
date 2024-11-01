package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"io"

	"compress/gzip"

	"github.com/Dkwkoaca/singtools/get"
	"github.com/Dkwkoaca/singtools/log"
	"github.com/sagernet/sing/common/json"
	bolt "go.etcd.io/bbolt"
)

type CacheManager struct {
	db      *bolt.DB
	path    string
	mu      sync.Mutex
	writeMu sync.Mutex
	temp    string
	logger  *log.Logger
}

// 添加常量定义
const (
	bucketURLData        = "URLData"
	bucketURLs           = "URLs"
	bucketUpdatedContent = "UpdatedContent"
	bucketProxies        = "Proxies" // 新增
)

// URLData 包含 URL 的内容、哈希值和更新时间
type URLData struct {
	URL     string
	Content []byte // 压缩后的内容
	Hash    [32]byte
	Updated time.Time
}

// NewCacheManager 创建新的缓存管理器实例
func NewCacheManager(dbPath string, logLevel string) (*CacheManager, error) {
	// 检查是否为 gzip 文件
	if isGzipFile(dbPath) {
		// 创建临时文件用于解压缩
		tmpFile := dbPath + ".tmp"

		// 打开 gzip 文件
		gzFile, err := os.Open(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open gzip file: %w", err)
		}
		defer gzFile.Close()

		// 创建 gzip reader
		gzReader, err := gzip.NewReader(gzFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		// 创建临时文件
		tmpDB, err := os.Create(tmpFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer tmpDB.Close()

		// 解压缩到临时文件
		_, err = io.Copy(tmpDB, gzReader)
		if err != nil {
			os.Remove(tmpFile)
			return nil, fmt.Errorf("failed to decompress file: %w", err)
		}

		// 使用临时文件创建数据库连接
		db, err := bolt.Open(tmpFile, 0600, &bolt.Options{Timeout: 1 * time.Second})
		if err != nil {
			os.Remove(tmpFile)
			return nil, fmt.Errorf("failed to open database: %w", err)
		}

		// 创建所需的 bucket
		err = createBuckets(db)
		if err != nil {
			db.Close()
			os.Remove(tmpFile)
			return nil, err
		}

		logger := log.NewLogger(logLevel)

		return &CacheManager{
			db:      db,
			path:    dbPath,
			mu:      sync.Mutex{},
			writeMu: sync.Mutex{},
			temp:    tmpFile,
			logger:  logger,
		}, nil
	}

	// 非压缩文件的正常处理
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = createBuckets(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &CacheManager{
		db:      db,
		path:    dbPath,
		mu:      sync.Mutex{},
		writeMu: sync.Mutex{},
	}, nil
}

// createBuckets 创建所需的 buckets
func createBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{
			bucketURLData,
			bucketURLs,
			bucketUpdatedContent,
			bucketProxies,
		} {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
}

// Close 关闭数据库并处理临时文件
func (manager *CacheManager) Close() error {
	err := manager.db.Close()
	if err != nil {
		return err
	}

	// 如果存在临时文件，需要压缩回原文件
	if manager.temp != "" {
		// 创建新的 gzip 文件
		gzFile, err := os.Create(manager.path)
		if err != nil {
			return fmt.Errorf("failed to create gzip file: %w", err)
		}
		defer gzFile.Close()

		// 创建 gzip writer
		gzWriter := gzip.NewWriter(gzFile)
		defer gzWriter.Close()

		// 读取临时文件
		tmpFile, err := os.Open(manager.temp)
		if err != nil {
			return fmt.Errorf("failed to open temp file: %w", err)
		}
		defer tmpFile.Close()

		// 压缩写入
		_, err = io.Copy(gzWriter, tmpFile)
		if err != nil {
			return fmt.Errorf("failed to compress file: %w", err)
		}

		// 删除临时文件
		os.Remove(manager.temp)
	}

	return nil
}

func (manager *CacheManager) Stats() bolt.Stats {
	return manager.db.Stats()
}

// StoreURLData 存储最新的 URL 数据，只保留最新的版本
func (manager *CacheManager) StoreURLData(data URLData) error {
	// 获取写入锁
	manager.writeMu.Lock()
	defer manager.writeMu.Unlock()

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

	// 如果数据不存在，尝试获取并存储
	if err != nil && err.Error() == fmt.Sprintf("no data found for URL: %s", url) {
		// 创建新的 HTTP 客户端
		client := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 10 * time.Second,
				DisableKeepAlives:   true,
			},
		}

		// 获取内容
		content, err := fetchContent(url, client)
		if err != nil {
			return URLData{}, fmt.Errorf("failed to fetch content: %w", err)
		}

		// 解析内容
		parsedData, err := get.ParseContent(string(content))
		if err != nil {
			return URLData{}, fmt.Errorf("failed to parse content: %w", err)
		}

		// 转换为 JSON
		contentStr, err := json.Marshal(parsedData)
		if err != nil {
			return URLData{}, fmt.Errorf("failed to marshal content: %w", err)
		}

		// 创建新的 URLData
		data = URLData{
			URL:     url,
			Content: contentStr,
			Hash:    hashContent(content),
			Updated: time.Now(),
		}

		// 存储数据
		if err := manager.StoreURLData(data); err != nil {
			return URLData{}, fmt.Errorf("failed to store URL data: %w", err)
		}

		return data, nil
	}

	if err != nil {
		return URLData{}, fmt.Errorf("failed to get URL data: %w", err)
	}

	// 解压缩内容
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

// 修改 getUpdateContent 方法
func (manager *CacheManager) getUpdateContent(urls []string) string {
	var content []interface{}
	urlData, err := manager.GetSubURLsData(urls)
	if err != nil {
		manager.logger.Error("Error getting URLs: " + err.Error())
		return ""
	}

	// 处理新的URL数据并存储到代理bucket中
	for _, v := range urlData {
		var data []interface{}
		err := json.Unmarshal(v.Content, &data)
		if err != nil {
			continue
		}
		content = append(content, data...)

		// 将新的代理数据存储到Proxies bucket中
		err = manager.mergeNewProxies(data)
		if err != nil {
			manager.logger.Error("Error merging new proxies: " + err.Error())
		}
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
			manager.logger.Error("Failed to log URL %s to URLs bucket: " + err.Error())
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
			manager.logger.Error("Failed to log updated content for URL %s: " + err.Error())
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
		manager.logger.Error("Failed to get updated content: " + err.Error())
		return ""
	}

	return string(content)
}

func (manager *CacheManager) mergeAllURLs() (string, error) {
	urlData, err := manager.GetAllURLData()
	if err != nil {
		manager.logger.Error("Error getting URLs: " + err.Error())
		return "", err
	}

	var all_node []interface{}
	for _, url := range urlData {
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
		manager.logger.Error("Error getting URLs: " + err.Error())
		return
	}
	var all_node []interface{}
	for _, url := range data {
		urls := url.URL
		context := url.Content

		var jsondata []interface{}
		err := json.Unmarshal(url.Content, &jsondata)
		if err != nil {
			manager.logger.Error(urls + " " + string(context))
		}

		lens := len(context)
		if lens > 200 {
			context = context[0:200]
		}

		all_node = append(all_node, jsondata...)
		fmt.Println("The url: ", urls, ", content: ", string(context), ", with length: ", lens, ", with nodes: ", len(jsondata))
	}
	unique := get.Unique(all_node)
	outbound := get.RemoveDuplicateOutbounds(unique)
	fmt.Println("Total:", len(data), "all nodes: ", len(all_node), ", unique nodes: ", len(unique), ", outbound nodes: ", len(outbound))
}

func (manager *CacheManager) UpdateURL(url string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   true,
		},
	}

	content, err := fetchContent(url, client)
	if err != nil {
		return fmt.Errorf("failed to fetch content: %w", err)
	}

	// 尝试解析内容
	data, err := get.ParseContent(string(content))
	if err != nil {
		if len(content) > 50 {
			manager.logger.Warn("Failed to parse content for URL " + url + ": " + err.Error() + "\n" + string(content)[0:50])
		} else {
			manager.logger.Warn("Failed to parse content for URL " + url + ": " + err.Error() + "\n" + string(content))
		}
		return nil
	}

	// 计算内容哈希
	hash := hashContent(content)

	// 存储URL数据
	contentStr, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	urlData := URLData{
		URL:     url,
		Content: contentStr,
		Hash:    hash,
		Updated: time.Now(),
	}

	// 只在实际写入数据时获取锁
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if err := manager.StoreURLData(urlData); err != nil {
		return fmt.Errorf("failed to store URL data: %w", err)
	}

	return nil
}
