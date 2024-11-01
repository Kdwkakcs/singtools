package cache

import (
	"fmt"
	"log"
	"sync"

	"github.com/Dkwkoaca/singtools/get"
)

// Read yaml config file to get the urls
func WriteUrlDB(yamlStr string, manager *CacheManager) {
	links, errr := get.ExtractLinks(yamlStr)
	if errr != nil {
		log.Println("Error extracting links:", errr)
		return
	}

	// get links from manager
	saved_links, err := manager.GetURLs()
	if err != nil {
		log.Println("Error getting saved links:", err)
	}

	links = append(links, saved_links...)
	links = get.Unique(links)
	log.Println("write url count:", len(links))

	// 创建一个带缓冲的通道来限制并发数
	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup

	// 使用channel来收集错误，避免锁的竞争
	errChan := make(chan error, len(links))

	for _, url := range links {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := manager.UpdateURL(url); err != nil {
				errChan <- fmt.Errorf("failed to update URL %s: %v", url, err)
			}
		}(url)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)

	// 收集错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// 在所有URL处理完成后，一次性获取锁并进行合并操作
	manager.writeMu.Lock()
	defer manager.writeMu.Unlock()

	// 合并内容
	content, err := manager.mergeAllURLs()
	if err != nil {
		log.Println("Error merging URLs:", err)
		return
	}

	err = manager.storeUpdatedContent(content)
	if err != nil {
		log.Println("Error storing merged content:", err)
	}

	// 报告错误统计
	if len(errs) > 0 {
		log.Printf("Completed with %d errors out of %d URLs\n", len(errs), len(links))
	}
}

func ReadUrlDB(yamlStr string, manager *CacheManager) {
}
