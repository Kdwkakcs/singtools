package main

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"io/ioutil"
// 	"os"
// 	"strings"

// 	// "github.com/spyzhov/ajson"
// 	"github.com/Dkwkoaca/singtools/parse"
// 	"github.com/Dkwkoaca/singtools/ping"
// 	"github.com/Dkwkoaca/singtools/utils"
// 	"github.com/sagernet/sing/service"
// 	"github.com/sagernet/sing/service/pause"
// 	"github.com/spf13/cobra"
// )

// func init() {
// 	var subCmd = &cobra.Command{
// 		Use:   "sub",
// 		Short: "Run the lite test on sub protocol",
// 		Run:   run_sub,
// 	}

// 	// Define flags for subCmd
// 	subCmd.PersistentFlags().StringVarP(&inputFilePath, "input", "i", "", "Input file path")
// 	subCmd.PersistentFlags().StringVarP(&outputFilePath, "output", "o", "out.json", "Output file path")
// 	subCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "", "Config file path")
// 	subCmd.PersistentFlags().BoolVarP(&detectCountry, "detect", "d", false, "Whether to detect country")
// 	subCmd.PersistentFlags().StringVarP(&outputCountry, "country", "t", "country.json", "Whether to output country")
// 	subCmd.PersistentFlags().BoolVarP(&downloadMMDB, "download", "l", false, "Whether to output country")
// 	subCmd.PersistentFlags().StringVarP(&level, "level", "e", "warn", "日志等级，可选值：trace debug info warn error fatal panic")
// 	subCmd.PersistentFlags().StringVarP(&hashFIles, "hashs", "s", "hash.json", "Hash file path")
// 	subCmd.PersistentFlags().BoolVarP(&RemoteIP, "remote", "r", false, "Whether to get remote ip")
// 	subCmd.PersistentFlags().StringVarP(&filter, "filter", "f", "tls, shadowsocks", "Filter node based on type, tls to force enable tls, split by ', '")

// 	subCmd.MarkPersistentFlagRequired("input")

// 	// Add subCmd to rootCmd
// 	mainCommand.AddCommand(subCmd)

// }

// func run_sub(cmd *cobra.Command, args []string) {
// 	fmt.Printf("Input file: %s\n", inputFilePath)
// 	fmt.Printf("Output file: %s\n", outputFilePath)
// 	fmt.Printf("Config file: %s\n", configFilePath)

// 	// parse config file
// 	testProfileOptions, err := parseConfigFile(configFilePath)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	fmt.Println("Filter based on:", testProfileOptions.Filter)
// 	detectCountrys := (detectCountry || testProfileOptions.Detect)
// 	fmt.Println(detectCountrys)
// 	if detectCountrys {
// 		if downloadMMDB {
// 			utils.CheckGeoLite2("GeoLite2-Country.mmdb")
// 		}
// 	}
// 	var options parse.BoxOptions
// 	if testProfileOptions.RemoveDup {
// 		file, err := os.Open(inputFilePath)
// 		if err != nil {
// 			panic(err)
// 		}
// 		defer file.Close()

// 		// read the file into a byte array
// 		byteValue, _ := ioutil.ReadAll(file)
// 		var result map[string]interface{}
// 		json.Unmarshal([]byte(byteValue), &result)
// 		result, err = utils.Reduplicates(result, hashFIles)
// 		outbounds, ok := result["outbounds"].([]map[string]interface{})
// 		if ok {
// 			filterOption := strings.Split(filter, ", ")
// 			// fmt.Println(filterOption)
// 			outbounds, err = parse.Filter(outbounds, filterOption)
// 			// fmt.Println(len(outbounds))
// 			result["outbounds"] = outbounds
// 		}
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}
// 		jsonss, err := json.MarshalIndent(result, "", "\t")
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}
// 		tmpfile := generateRandomFileName()
// 		ioutil.WriteFile(tmpfile, jsonss, 0644)

// 		configPaths := []string{tmpfile}
// 		options, err = parse.ReadConfigs(configPaths)
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}
// 		options.Log.Level = level
// 		os.Remove(tmpfile)
// 	} else {
// 		// create []string for configPaths
// 		configPaths := []string{inputFilePath}
// 		options, err = parse.ReadConfigs(configPaths)
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}
// 	}

// 	options.Log.Level = level
// 	// filter options
// 	options = filterOptions(options)
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
// 	ctx = service.ContextWithDefaultRegistry(ctx)
// 	ctx = pause.ContextWithDefaultManager(ctx)

// 	outbounds, err := createOutbounds(ctx, options)
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// 	p := ping.ProfileTest{
// 		Outbound: outbounds,
// 		Options:  &testProfileOptions,
// 		RemoteIP: RemoteIP,
// 	}

// 	fmt.Println("Start testing...")
// 	nodes, err := p.TestAllGrouped(ctx)

// 	// fmt.Println(nodes)

// 	// write filtered nodes to output file
// 	jsonFile, err := os.Open(inputFilePath)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer jsonFile.Close()
// 	// read our opened jsonFile as a byte array.
// 	byteValue, _ := io.ReadAll(jsonFile)
// 	filteredJson, err := filterJson(byteValue, nodes, p.Options.Filter)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	// write to output file
// 	ioutil.WriteFile(outputFilePath, filteredJson, 0644)

// 	// write country to output file
// 	cjson, err := json.MarshalIndent(nodes, "", "\t")
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	ioutil.WriteFile(outputCountry, cjson, 0644)

// }
