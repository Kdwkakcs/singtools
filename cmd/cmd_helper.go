package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Dkwkoaca/singtools/parse"
	"github.com/Dkwkoaca/singtools/ping"

	// "github.com/spyzhov/ajson"
	"github.com/sagernet/sing-box/adapter"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/outbound"
	"github.com/sagernet/sing-box/route"
	"github.com/sagernet/sing/common"
	F "github.com/sagernet/sing/common/format"
)

var (
	inputFilePath  string
	outputFilePath string
	configFilePath string
	detectCountry  bool
	outputCountry  string
	downloadMMDB   bool
	level          string
	hashFIles      string
	RemoteIP       bool
	filter         string
)

func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true // Found the value
		}
	}
	return false // Value not found
}

// define a function to filter json
func filterJson(jsons []byte, res []ping.Node, filters string) ([]byte, error) {
	var sucessNode []string
	for _, re := range res {
		switch {
		case filters == "pingonly":
			// fmt.Println("Filter based on ping;")
			if re.Ping > 0 {
				sucessNode = append(sucessNode, re.Tag)
			}
		case filters == "speedonly":
			// fmt.Println("Filter based on speedonly;")
			if re.AvgSpeed > 0 {
				sucessNode = append(sucessNode, re.Tag)
			}
		case filters == "all":
			// fmt.Println("Filter based on all;")
			if re.Ping > 0 && re.AvgSpeed > 0 {
				sucessNode = append(sucessNode, re.Tag)
			}
		default:
			if re.Ping > 0 {
				sucessNode = append(sucessNode, re.Tag)
			}
		}
	}
	// print all node number, sucess node number and the rate
	fmt.Println("All node number: ", len(res))
	fmt.Println("Sucess node number: ", len(sucessNode))
	fmt.Println("Rate: ", float64(len(sucessNode))/float64(len(res)))
	// convert json to map
	var result map[string]interface{}
	json.Unmarshal(jsons, &result)
	// filter json
	outbounds := result["outbounds"].([]interface{})
	filteredOutbounds := []interface{}{}
	for _, outbound := range outbounds {
		if tag, ok := outbound.(map[string]interface{})["tag"].(string); ok {
			if Contains(sucessNode, tag) {
				filteredOutbounds = append(filteredOutbounds, outbound)
			}
		}
	}
	fmt.Println("Output outbounds  number: ", len(filteredOutbounds))
	result["outbounds"] = filteredOutbounds
	// convert map to json
	jsonss, err := json.MarshalIndent(result, "", "\t")
	return jsonss, err
}

func createOutbounds(ctx context.Context, options parse.BoxOptions) ([]adapter.Outbound, error) { // 替换 Options 为你的实际类型
	var defaultLogWriter io.Writer
	if options.PlatformInterface != nil {
		defaultLogWriter = io.Discard
	}
	logFactory, err := log.New(log.Options{
		Context:        ctx,
		Options:        common.PtrValueOrDefault(options.Log),
		Observable:     true,
		DefaultWriter:  defaultLogWriter,
		BaseTime:       time.Now(),
		PlatformWriter: options.PlatformLogWriter,
	})
	if err != nil {
		return nil, err
	}
	router, err := route.NewRouter(
		ctx,
		logFactory,
		common.PtrValueOrDefault(options.Route),
		common.PtrValueOrDefault(options.DNS),
		common.PtrValueOrDefault(options.NTP),
		options.Inbounds,
		nil,
	)
	if err != nil {
		return nil, err
	}
	outbounds := make([]adapter.Outbound, 0, len(options.Outbounds))
	var lastErr error
	var lastErrDesc string
	var skiptype = []string{"selector", "block", "dns", "urltest", "tor"}
	for i, outboundOptions := range options.Outbounds {
		var out adapter.Outbound
		var tag string
		if outboundOptions.Tag != "" {
			tag = outboundOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		if Contains(skiptype, outboundOptions.Type) {
			continue
		}
		out, err = outbound.New(
			ctx,
			router,
			logFactory.NewLogger(F.ToString("outbound/", outboundOptions.Type, "[", tag, "]")),
			tag,
			outboundOptions)

		if err != nil {
			lastErrDesc = fmt.Sprintf("parse outbound[%d] \t%+v \terror: %+v", i, out, err)
			log := logFactory.NewLogger(F.ToString("outbound/", outboundOptions.Type, "[", tag, "]"))
			log.Error(lastErrDesc)
			lastErr = err
			continue
		}
		outbounds = append(outbounds, out)
	}
	if lastErr != nil {
		// return nil, fmt.Errorf("error creating outbounds: %w", lastErr)
		fmt.Println("Last err", lastErr)
	}
	return outbounds, nil
}

// define a function to parse config file
func parseConfigFile(configFilePath string) (ping.Config, error) {
	var options ping.Config
	configFile, err := os.Open(configFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer configFile.Close()
	byteValue, _ := io.ReadAll(configFile)
	if err := json.Unmarshal(byteValue, &options); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if options.Timeout != 0 {
		options.Timeout = options.Timeout * time.Second
	}
	if detectCountry {
		options.Detect = true
	}
	return options, nil
}

func generateRandomFileName() string {
	byteSlice := make([]byte, 10) // 10 bytes, adjust as needed
	_, err := rand.Read(byteSlice)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(byteSlice) // returns a hexadecimal representation of the random bytes
}

func filterOptions(options parse.BoxOptions) parse.BoxOptions {
	var filteredOptions parse.BoxOptions
	filteredOptions.Inbounds = options.Inbounds
	filteredOptions.Outbounds = options.Outbounds
	filteredOptions.Log = options.Log
	return filteredOptions
}
