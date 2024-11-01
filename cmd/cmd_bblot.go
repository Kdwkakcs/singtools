package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/Dkwkoaca/singtools/cache"
	"github.com/spf13/cobra"
)

var bblotCommand = &cobra.Command{
	Use:   "bblot",
	Short: "Test bblot functionality with cache manager",
	Long:  `Test bblot functionality with cache manager and config file`,
	Run:   runBblot,
}

var (
	configFile string
	cacheFile  string
)

func init() {
	bblotCommand.Flags().StringVarP(&configFile, "config", "c", "", "config yml file path")
	bblotCommand.Flags().StringVarP(&cacheFile, "cache", "d", "./cache_all_url.db", "cache db file path")

	// 添加到主命令
	mainCommand.AddCommand(bblotCommand)
}

func runBblot(cmd *cobra.Command, args []string) {
	if configFile == "" {
		cmd.Help()
		return
	}

	manager, err := cache.NewCacheManager(cacheFile)
	if err != nil {
		log.Println("Error creating cache manager:", err)
		return
	}
	defer manager.Close()

	yml, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("Error reading config file: %v\n", err)
		return
	}

	cache.WriteUrlDB(string(yml), manager)

	urls, err := manager.GetURLs()
	if err != nil {
		log.Printf("Error getting URLs: %v\n", err)
		return
	}
	fmt.Println(strings.Join(urls, "\n"))
	manager.Glimpse()
	// str := manager.GetUpdated()
	// fmt.Println(str)
}
