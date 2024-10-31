package cache

import (
	"log"

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
		// 不要return，继续处理新的links
	}
	
	links = append(links, saved_links...)
	links = get.Unique(links)
	log.Println("write url count:", len(links))
	
	// 更新URLs，但不要因为单个URL的错误而中断整个过程
	errs := make([]error, 0)
	for _, url := range links {
		if err := manager.UpdateURL(url); err != nil {
			log.Printf("Warning: Failed to update URL %s: %v\n", url, err)
			errs = append(errs, err)
			continue
		}
	}
	
	// 即使有些URL更新失败，仍然生成合并内容
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
