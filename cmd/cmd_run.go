package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	// "github.com/spyzhov/ajson"
	"github.com/Dkwkoaca/singtools/parse"
	"github.com/Dkwkoaca/singtools/ping"
	"github.com/Dkwkoaca/singtools/utils"
	s "github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
	"github.com/spf13/cobra"
)

func init() {
	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "Run the lite test",
		Run:   run,
	}

	// Define flags for testCmd
	testCmd.PersistentFlags().StringVarP(&inputFilePath, "input", "i", "", "Input file path")
	testCmd.PersistentFlags().StringVarP(&outputFilePath, "output", "o", "out.json", "Output file path")
	testCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "", "Config file path")
	testCmd.PersistentFlags().BoolVarP(&detectCountry, "detect", "d", false, "Whether to detect country")
	testCmd.PersistentFlags().StringVarP(&outputCountry, "country", "t", "country.json", "Whether to output country")
	testCmd.PersistentFlags().BoolVarP(&downloadMMDB, "download", "l", false, "Whether to output country")
	testCmd.PersistentFlags().StringVarP(&level, "level", "e", "warn", "日志等级，可选值：trace debug info warn error fatal panic")
	testCmd.PersistentFlags().StringVarP(&hashFIles, "hashs", "s", "hash.json", "Hash file path")
	testCmd.PersistentFlags().BoolVarP(&RemoteIP, "remote", "r", false, "Whether to get remote ip")
	testCmd.PersistentFlags().StringVarP(&filter, "filter", "f", "tls, shadowsocks", "Filter node based on type, tls to force enable tls, split by ', '")

	testCmd.MarkPersistentFlagRequired("input")

	// Add testCmd to rootCmd
	mainCommand.AddCommand(testCmd)

}

func run(cmd *cobra.Command, args []string) {
	fmt.Printf("Input file: %s\n", inputFilePath)
	fmt.Printf("Output file: %s\n", outputFilePath)
	fmt.Printf("Config file: %s\n", configFilePath)

	// parse config file
	testProfileOptions, err := parseConfigFile(configFilePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Filter based on:", testProfileOptions.Filter)

	// 创建服务
	service, err := ping.NewService(&testProfileOptions)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 解析配置文件
	var options parse.BoxOptions
	if testProfileOptions.RemoveDup {
		file, err := os.Open(inputFilePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		// read the file into a byte array
		byteValue, _ := ioutil.ReadAll(file)
		var result map[string]interface{}
		json.Unmarshal([]byte(byteValue), &result)
		result, err = utils.Reduplicates(result, hashFIles)
		outbounds, ok := result["outbounds"].([]map[string]interface{})
		if ok {
			filterOption := strings.Split(filter, ", ")
			// fmt.Println(filterOption)
			outbounds, err = parse.Filter(outbounds, filterOption)
			// fmt.Println(len(outbounds))
			result["outbounds"] = outbounds
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		jsonss, err := json.MarshalIndent(result, "", "\t")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		tmpfile := generateRandomFileName()
		ioutil.WriteFile(tmpfile, jsonss, 0644)

		configPaths := []string{tmpfile}
		options, err = parse.ReadConfigs(configPaths)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		options.Log.Level = level
		os.Remove(tmpfile)
	} else {
		// create []string for configPaths
		configPaths := []string{inputFilePath}
		options, err = parse.ReadConfigs(configPaths)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	options.Log.Level = level
	// filter options
	options = filterOptions(options)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = s.ContextWithDefaultRegistry(ctx)
	ctx = pause.ContextWithDefaultManager(ctx)

	// 创建出站
	outbounds, err := createOutbounds(ctx, options)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Start testing...")
	// 运行测试
	nodes, err := service.RunTests(ctx, outbounds)
	if err != nil {
		// 修改这里：即使有错误也继续处理已获得的结果
		fmt.Printf("Warning: Some tests failed: %v\n", err)
		// 不要 os.Exit(1)
	}

	// write filtered nodes to output file
	jsonFile, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()
	// read our opened jsonFile as a byte array.
	byteValue, _ := io.ReadAll(jsonFile)
	filteredJson, err := filterJson(byteValue, nodes, testProfileOptions.Filter)
	if err != nil {
		fmt.Println(err)
	}
	// write to output file
	ioutil.WriteFile(outputFilePath, filteredJson, 0644)

	// write country to output file
	cjson, err := json.MarshalIndent(nodes, "", "\t")
	if err != nil {
		fmt.Println(err)
	}
	ioutil.WriteFile(outputCountry, cjson, 0644)

}
