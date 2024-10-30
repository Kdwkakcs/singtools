package cache

import (
	"log"

	"github.com/Dkwkoaca/singtools/get"
)

// Read yaml config file to get the urls
func WriteUrlDB(yamlStr string, manager *CacheManager) {
	links, errr := get.ExtractLinks(yamlStr)
	// get links from manager
	saved_links, err := manager.GetURLs()
	if err != nil {
		log.Println(err)
	}
	links = append(links, saved_links...)
	links = get.Unique(links)
	if errr != nil {
		log.Println(errr)
	}
	links = get.Unique(links)
	log.Println("write url count:", len(links))
	manager.UpdateURLs(links)
}

func ReadUrlDB(yamlStr string, manager *CacheManager) {
}
