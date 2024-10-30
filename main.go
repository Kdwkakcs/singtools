package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Dkwkoaca/singtools/parse"
	"github.com/Dkwkoaca/singtools/ping"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/outbound"
	"github.com/sagernet/sing-box/route"
	"github.com/sagernet/sing/common"
	F "github.com/sagernet/sing/common/format"
	s "github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
	"github.com/spf13/cobra"
)

var (
	inputFilePath  string
	outputFilePath string
	configFilePath string
	detectCountry  bool
	outputCountry  string
	downloadMMDB   bool
	level          string
	hashFiles      string
	remoteIP       bool
	filter         string
)

func run(cmd *cobra.Command, args []string) {
	// 1. 解析配置文件
	configBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}

	// 2. 创建配置对象
	var config ping.Config
	if err := json.Unmarshal(configBytes, &config); err != nil {
		fmt.Printf("Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// 3. 应用命令行参数覆盖配置
	// 将 timeout 从数字转换为 Duration
	// config.InitWithDefaults()
	// 应用命令行参数覆盖配置
	fmt.Printf("Parsed config: %+v\n", config)
	config.Detect = config.Detect || detectCountry
	config.RemoteIP = remoteIP
	// if downloadMMDB || (config.Detect && !utils.FileExists(config.GeoIPDBPath)) {
	// 	fmt.Println("Downloading GeoIP database...")
	// 	if err := utils.DownloadGeoLite2(config.GeoIPDBPath); err != nil {
	// 		fmt.Printf("Error downloading GeoIP database: %v\n", err)
	// 		os.Exit(1)
	// 	}
	// }

	// 验证配置
	// if err := config.Validate(); err != nil {
	// 	fmt.Printf("Invalid configuration: %v\n", err)
	// 	os.Exit(1)
	// }

	// 3. 创建测试服务
	service, err := ping.NewService(&config)
	if err != nil {
		fmt.Printf("Error creating service: %v\n", err)
		os.Exit(1)
	}
	defer service.Close()

	// 4. 读取并解析输入文件
	inputBytes, err := os.ReadFile(inputFilePath)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	// 5. 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 6. 创建出站连接
	configPaths := []string{inputFilePath}
	options, err := parse.ReadConfigs(configPaths)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	options.Log.Level = level
	// filter options
	options = filterOptions(options)
	ctxs, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctxs = s.ContextWithDefaultRegistry(ctxs)
	ctxs = pause.ContextWithDefaultManager(ctxs)
	outbounds, err := createOutbounds(ctxs, options)

	if err != nil {
		fmt.Printf("Error creating outbounds: %v\n", err)
		os.Exit(1)
	}

	// 7. 执行测试
	fmt.Println("Starting tests...")
	nodes, err := service.RunTests(ctx, outbounds)
	if err != nil {
		// 修改这里：即使有错误也继续处理已获得的结果
		fmt.Printf("Warning: Some tests failed: %v\n", err)
		// 不要 os.Exit(1)
	}

	// 8. 处理测试结果
	if len(nodes) > 0 {
		// 8.1 写入过滤后的配置
		filteredJson, err := filterJson(inputBytes, nodes, config.SpeedTestMode)
		if err != nil {
			fmt.Printf("Error filtering results: %v\n", err)
		} else {
			if err := os.WriteFile(outputFilePath, filteredJson, 0644); err != nil {
				fmt.Printf("Error writing output file: %v\n", err)
			}
		}

		// 8.2 写入国家信息
		if config.Detect {
			countryJson, err := json.MarshalIndent(nodes, "", "    ")
			if err != nil {
				fmt.Printf("Error marshaling country info: %v\n", err)
			} else {
				if err := os.WriteFile(outputCountry, countryJson, 0644); err != nil {
					fmt.Printf("Error writing country file: %v\n", err)
				}
			}
		}
	}

	// 9. 输出统计信息
	fmt.Printf("\nTest Summary:\n")
	fmt.Printf("Total Nodes: %d\n", len(outbounds))
	fmt.Printf("Successful Nodes: %d\n", len(nodes))
	fmt.Printf("Success Rate: %.2f%%\n", float64(len(nodes))/float64(len(outbounds))*100)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "speedtest",
		Short: "Test speed of sing-box outbounds",
		Run:   run,
	}

	// 设置命令行参数
	rootCmd.PersistentFlags().StringVarP(&inputFilePath, "input", "i", "", "Input configuration file path")
	rootCmd.PersistentFlags().StringVarP(&outputFilePath, "output", "o", "out.json", "Output file path")
	rootCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "", "Test configuration file path")
	rootCmd.PersistentFlags().BoolVarP(&detectCountry, "detect", "d", false, "Enable country detection")
	rootCmd.PersistentFlags().StringVarP(&outputCountry, "country", "t", "country.json", "Country information output file")
	rootCmd.PersistentFlags().BoolVarP(&downloadMMDB, "download", "l", false, "Download GeoLite2 database")
	rootCmd.PersistentFlags().StringVarP(&level, "level", "e", "warn", "Log level (trace/debug/info/warn/error/fatal/panic)")
	rootCmd.PersistentFlags().StringVarP(&hashFiles, "hashs", "s", "hash.json", "Hash file path")
	rootCmd.PersistentFlags().BoolVarP(&remoteIP, "remote", "r", false, "Enable remote IP detection")
	rootCmd.PersistentFlags().StringVarP(&filter, "filter", "f", "tls, shadowsocks", "Node type filter (comma separated)")

	// 设置必需的参数
	rootCmd.MarkPersistentFlagRequired("input")
	rootCmd.MarkPersistentFlagRequired("config")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// // createOutbounds 辅助函数，用于创建出站连接
// func createOutbounds(ctx context.Context, configBytes []byte, logLevel string) ([]adapter.Outbound, error) {
// 	var options parse.BoxOptions
// 	if err := json.Unmarshal(configBytes, &options); err != nil {
// 		return nil, fmt.Errorf("parse config error: %w", err)
// 	}

//		options.Log.Level = logLevel
//		return parse.CreateOutbounds(ctx, options)
//	}
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

func Contains[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true // Found the value
		}
	}
	return false // Value not found
}

func filterOptions(options parse.BoxOptions) parse.BoxOptions {
	var filteredOptions parse.BoxOptions
	filteredOptions.Inbounds = options.Inbounds
	filteredOptions.Outbounds = options.Outbounds
	filteredOptions.Log = options.Log
	return filteredOptions
}
