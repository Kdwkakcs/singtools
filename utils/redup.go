package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strconv"
)

func generateHash(item map[string]interface{}) string {
	itemCopy := make(map[string]interface{})
	keys := make([]string, 0, len(item))

	for k := range item {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if k != "tag" {
			itemCopy[k] = item[k]
		}
	}

	itemJSON, _ := json.Marshal(itemCopy)
	// fmt.Println(string(itemJSON))
	hasher := md5.New()
	hasher.Write(itemJSON)
	return hex.EncodeToString(hasher.Sum(nil))
}

func removeDuplicatesOptimized(jsonData []interface{}, files string) []interface{} {
	seenHashes := make(map[string]bool, len(jsonData))
	var uniqueJSON []interface{}
	for _, item := range jsonData {
		itemHash := generateHash(item.(map[string]interface{})) // Perform type assertion
		if !seenHashes[itemHash] {
			seenHashes[itemHash] = true
			uniqueJSON = append(uniqueJSON, item)
		}
	}
	// save seenHashes to file
	jsons, err := json.Marshal(seenHashes)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	err = ioutil.WriteFile(files, jsons, 0644)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	return uniqueJSON
}

func Reduplicates(result map[string]interface{}, files string) (map[string]interface{}, error) {
	outbounds, ok := result["outbounds"].([]interface{})
	if !ok {
		fmt.Println("Error: invalid type for jsons[\"outbounds\"]", result["outbounds"])
		return nil, fmt.Errorf("invalid type for jsons[\"outbounds\"]")
	}
	// outbounds = filter(outbounds)
	var ress []map[string]interface{}
	for i, value := range outbounds {
		outbound, ok := value.(map[string]interface{})
		if !ok {
			fmt.Println("Error: invalid type for outbounds[", i, "]", value)
			return nil, fmt.Errorf("invalid type for outbounds[%d]", i)
		}
		val, exists := outbound["tag"]
		if !exists || val == "" {
			outbound["tag"] = "proxy" + strconv.Itoa(i)
		}
		ress = append(ress, outbound)
	}
	fmt.Println("Before remove duplicates:", len(outbounds))
	ressInterface := make([]interface{}, len(ress))
	for i, v := range ress {
		ressInterface[i] = v
	}
	ressInterface = removeDuplicatesOptimized(ressInterface, files)
	ress = make([]map[string]interface{}, len(ressInterface))
	for i, v := range ressInterface {
		ress[i] = v.(map[string]interface{})
	}
	fmt.Println("After remove duplicates:", len(ress))
	result["outbounds"] = ress
	return result, nil
}
