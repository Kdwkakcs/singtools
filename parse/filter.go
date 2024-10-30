package parse

import (
	"fmt"
)

// Contains checks if a string is present in a slice.
func Contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// Filter filters out maps based on tags and a specific "tls" key presence when required.
func Filter(out []map[string]interface{}, tags []string) ([]map[string]interface{}, error) {
	if len(out) == 0 {
		return nil, fmt.Errorf("empty input slice")
	}

	var result []map[string]interface{}
	tlsRequired := Contains(tags, "tls") // Check if TLS is a required tag

	for _, mapItem := range out {
		tagValue, ok := mapItem["type"].(string) // Ensure the tag value is a string
		if ok && Contains(tags, tagValue) {
			continue // Skip if the current map's tag is in the tags slice
		}

		if tlsRequired && !ContainsKey(mapItem, "tls") {
			continue // Skip if TLS is required but the map doesn't have a "tls" key
		}

		result = append(result, mapItem) // Add map to result if it passes all checks
	}

	return result, nil
}

// ContainsKey checks if a map contains a specific key.
func ContainsKey(m map[string]interface{}, key string) bool {
	_, exists := m[key]
	return exists
}
