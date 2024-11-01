package get

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unsafe"

	"github.com/levigross/grequests"
	"github.com/sagernet/sing-box/option"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

/*
#cgo LDFLAGS: -L./ -lsubconverter -static -lpcre2-8 -lyaml-cpp -lstdc++ -lm
#include <stdlib.h>

// 声明外部 C 函数
char* NodeToSingbox(const char* input, const char* delimiter);
*/
import "C"

func NodeToSingbox(input string, delimiter string) string {
	// Convert Go string to C string
	cInput := C.CString(input)
	defer C.free(unsafe.Pointer(cInput)) // Enasure to free Go memory allocated by C.CString
	cDelimiter := C.CString(delimiter)
	defer C.free(unsafe.Pointer(cDelimiter))

	// Call C++ function to process the string
	cOutput := C.NodeToSingbox(cInput, cDelimiter)
	if cOutput == nil {
		return ""
	}

	// Convert C string to Go string
	goOutput := C.GoString(cOutput)

	// release memory allocated by C++ code
	C.free(unsafe.Pointer(cOutput))

	return goOutput
}

func ParseUrl(url string) ([]interface{}, error) {
	context, err := GetUrl(url)
	if err != nil {
		return nil, err
	}
	return ParseContent(context)
}

func ParseContent(context string) ([]interface{}, error) {
	// check if the context contains: vless vmess trojan ss ssr hysteria2 hysteria1
	prefix := []string{"vmess:/", "vless:/", "trojan:/", "ss:/", "ssr:/", "hysteria2:/", "hysteria:/", "hy2:/", "hy:/", "tuic:/"}
	prefixRegex := "(?m)^(" + strings.Join(prefix, "|") + ").*"
	re := regexp.MustCompile(prefixRegex)
	var nodes string
	if ContainsPrefix(context, prefix) {
		// find all the proxies and base64 encode them
		proxies := re.FindAllString(context, -1)
		proxiesStr := strings.Join(proxies, "\n")
		proxiesStr = base64.StdEncoding.EncodeToString([]byte(proxiesStr))
		nodes = NodeToSingbox(proxiesStr, "multi")
		if nodes == "[]" {
			return nil, fmt.Errorf("no proxies found")
		}
	} else {
		nodes = NodeToSingbox(context, "multi")
		if nodes == "[]" {
			data, types, err := getContentType(context)
			if err != nil {
				return nil, err
			}
			if types == "json" {
				return convertToSingbox(data), nil
			} else if types == "yaml" {
				return convertToYaml(data), nil
			}
			return nil, nil
		}
	}
	var jsondata []interface{}
	err := json.Unmarshal([]byte(nodes), &jsondata)
	if err != nil {
		return nil, err
	}
	return jsondata, nil
}
func getContentType(context string) ([]interface{}, string, error) {
	// fmt.Println(context)
	// test if the context is a json
	var jsondata []interface{}
	err := json.Unmarshal([]byte(context), &jsondata)
	if err == nil {
		return jsondata, "json", nil
	}

	// test if the context is a yaml
	v := viper.New()
	v.SetConfigType("yaml")
	err = v.ReadConfig(bytes.NewBuffer([]byte(context)))
	if err != nil {
		return nil, "", err
	}
	proxies := v.Get("proxies")
	if proxiesss, ok := proxies.([]interface{}); ok {
		return proxiesss, "yaml", nil
	}

	return nil, "", fmt.Errorf("proxies not found")
}

func convertToYaml(data []interface{}) []interface{} {
	var res []interface{}
	var outbound []interface{}
	for _, d := range data {
		// convert to yaml
		str, err := yaml.Marshal(&d)
		if err != nil {
			continue
		}
		proxy := NodeToSingbox(string(str), "clash")
		err = json.Unmarshal([]byte(proxy), &outbound)
		if err != nil {
			continue
		}
		res = append(res, outbound[0])
	}
	return res
}

func convertToSingbox(data []interface{}) []interface{} {
	// convert to yaml
	var res []interface{}
	var outbound []interface{}
	for _, d := range data {
		str, err := json.Marshal(&d)
		if err != nil {
			continue
		}
		proxy := NodeToSingbox(string(str), "singbox")
		err = json.Unmarshal([]byte(proxy), &outbound)
		if err != nil {
			continue
		}
		res = append(res, outbound[0])
	}
	return res
}

func GetUrl(url string) (string, error) {
	ro := &grequests.RequestOptions{
		RequestTimeout: time.Second * 10,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
		},
	}
	resp, err := grequests.Get(url, ro)
	if err != nil {
		return "", err
	}
	return resp.String(), nil
}

func CheckOutbound(res *[]interface{}) []interface{} {
	var returnOutbound []interface{}
	var outbound option.Outbound
	// var jsonOutbound []interface{}
	for _, outs := range *res {
		outsMap := outs.(map[string]interface{})
		// err := outbound.UnmarshalJSON([]byte(outsMap))
		modifiedJson, err := json.Marshal(outsMap)
		if err != nil {
			// fmt.Println(err, " and the map is", outsMap)
			continue
		}
		err = outbound.UnmarshalJSON(modifiedJson)
		if err != nil {
			// fmt.Println(err, " and the map is", outsMap)
			continue
		}
		// fmt.Println(string(modifiedJson))
		returnOutbound = append(returnOutbound, outs)
	}

	return returnOutbound
}
