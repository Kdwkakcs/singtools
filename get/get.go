package get

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/levigross/grequests"
)

var (
	category string
)

type GetConfig struct {
	InputFile  string
	OutputFile string
	Category   string
	SaveFile   string
}

// func main() {
// 	rootCmd := &cobra.Command{
// 		Use:   "collect",
// 		Short: "Collect links and write to output file",
// 		Run:   collectLinks,
// 	}

// 	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "input config file")
// 	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")
// 	rootCmd.Flags().StringVarP(&category, "category", "t", "tg", "output file path")
// 	rootCmd.Flags().StringVarP(&saveFile, "save", "s", "", "file to save the links") // Added command parameter to specify the file to save the links

// 	if err := rootCmd.Execute(); err != nil {
// 		log.Fatal(err)
// 	}
// }

func GetProxies(args GetConfig) {
	collectLinks(args)
}

func collectLinks(args GetConfig) {
	// args.category to category var
	category = args.Category
	ymlString, err := ioutil.ReadFile(args.InputFile)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	links, _ := ExtractLinks(string(ymlString))
	links = Unique(links)

	// write all links to output file
	err = ioutil.WriteFile("saved_links.txt", []byte(strings.Join(links, "\n")), 0644)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	sucessLinks := make([]string, 0, len(links))
	log.Println("collect links count:", len(links))
	ro := &grequests.RequestOptions{
		RequestTimeout: time.Second * 100,
		DialTimeout:    time.Second * 100,
	}
	// test the grequests can get the content?
	// 'vmess', 'vless', 'trojan', 'ss', 'ssr', 'hysteria2', "hysteria1"
	prefix := []string{"vmess:/", "vless:/", "trojan:/", "ss:/", "ssr:/", "hysteria2:/", "hysteria:/", "hy2:/", "hy:/", "tuic:/"}
	res := make([]interface{}, 0, 50000) // Use a slice with initial capacity

	prefixRegex := "(?m)^(" + strings.Join(prefix, "|") + ").*"
	re := regexp.MustCompile(prefixRegex)
	urlList := make([]string, 0, 5000)

	var wg sync.WaitGroup
	var mu sync.Mutex                        // Mutex for synchronization
	goroutinePool := make(chan struct{}, 10) // Create a buffered channel with a capacity of 10

	log.Println("collect links count: ", len(links))
	for _, link := range links {
		wg.Add(1)

		goroutinePool <- struct{}{} // Acquire a spot

		go func(link string) {
			// log.Println("collecting:", link)
			defer wg.Done()
			defer func() { <-goroutinePool }() // Release the spot

			resp, err := grequests.Get(link, ro)
			if err != nil {
				// log.Println("Error accessing:", link, "Error:", err)
				return
			}
			defer resp.Close()

			// _, ok := isHTML(*resp)
			// if ok {
			// 	log.Println("Error accessing:", link, "Error: is HTML")
			// 	return
			// } else {
			if true {
				strs := resp.String()
				// log.Println("collecting:", strs)
				if found := ContainsPrefix(strs, prefix); found {
					subs := re.FindAllString(strs, -1)
					// log.Println("sub count:", len(subs))
					mu.Lock()
					urlList = append(urlList, subs...)
					sucessLinks = append(sucessLinks, link)
					mu.Unlock()
				} else {
					// url := "http://0.0.0.0:25500/sub?target=singbox&url=" + link
					// log.Println("collecting:", link)
					subs, err := ParseUrl(link)
					if err != nil {
						// log.Println("Error parsing:", link, "Error:", err)
						return
					} else {
						mu.Lock()
						res = append(res, subs...)
						sucessLinks = append(sucessLinks, link)
						mu.Unlock()
					}
				}
			}
		}(link)
	}

	wg.Wait() // Wait for all requests to complete

	// write the url to a localfile and convert into clash file
	outputData := strings.Join(urlList, "\n")
	encodedData := base64.StdEncoding.EncodeToString([]byte(outputData))

	subs := NodeToSingbox(encodedData, "mutli")
	var base64Data []interface{}
	err = json.Unmarshal([]byte(subs), &base64Data)
	if err != nil {
		log.Println("Error parsing base64 context. ", "Error:", err)
	} else {
		mu.Lock()
		res = append(res, base64Data...)
		mu.Unlock()
	}

	uniqueRes := Unique(res)
	uniqueRes = CheckOutbound(&uniqueRes)
	log.Println("All count of collected node :", len(res))
	log.Println("Removed same nodes", len(uniqueRes))
	// log.Println(uniqueRes)

	// Reduplicate and rename

	uniqueRes = RemoveDuplicateOutbounds(uniqueRes)
	uniqueRes = RenameOutbounds(uniqueRes)
	log.Println("All count of unique node :", len(uniqueRes))

	ress := map[string]interface{}{
		"log":       map[string]interface{}{},
		"dns":       map[string]interface{}{},
		"ntp":       map[string]interface{}{},
		"inbounds":  []interface{}{},
		"outbounds": uniqueRes,
		"route":     map[string]interface{}{},
	}

	outputBytes, err := json.MarshalIndent(ress, "", "  ")
	if err != nil {
		log.Fatalf("error marshaling uniqueRes: %v", err)
	}

	err = ioutil.WriteFile(args.OutputFile, outputBytes, 0644)
	if err != nil {
		log.Fatalf("error writing output file: %v", err)
	}

	// Added code to save the links to a file
	log.Println("sucess links count: ", len(sucessLinks))
	sort.Strings(sucessLinks)
	if args.SaveFile != "" {
		err = ioutil.WriteFile(args.SaveFile, []byte(strings.Join(sucessLinks, "\n")), 0644)
		if err != nil {
			log.Fatalf("error writing save file: %v", err)
		}
	}
}
