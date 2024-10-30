package cache

import (
	"log"

	"github.com/Dkwkoaca/singtools/get"
)

// Read yaml config file to get the urls
func WriteUrlDB(yamlStr string, manager *CacheManager) {
	links, errr:= get.ExtractLinks(yamlStr)
	if errr != nil {
		log.Println(errr)
	}
	links = get.Unique(links)
	log.Println("write url count:", len(links))
	manager.UpdateURLs(links)
}

func ReadUrlDB(yamlStr string, manager *CacheManager) {
}
