package ping

import (
	"fmt"
	"time"
)

const (
	// TestMode 测试模式
	SpeedOnly = "speedonly"
	PingOnly  = "pingonly"
	AllTest   = "all"

	// TestType 测试类型
	TypeAll    = iota // 全部测试
	TypeRetest        // 重新测试

	// Defaults 默认值
	DefaultRetries = 2

	// Limits 限制值
	MaxProtocolNodes = 40
	MaxTotalNodes    = 200

	// URLs 默认URL

	// HTTP Headers
	DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
)

// 定义默认值常量
const (
	// 基本配置默认值
	DefaultGroupName     = "Default"
	DefaultSpeedTestMode = "all" // "all", "pingonly", "speedonly"
	DefaultPingURL       = "https://www.gstatic.com/generate_204"
	DefaultDownloadURL   = "https://download.microsoft.com/download/2/7/A/27AF1BE6-DD20-4CB4-B154-EBAB8A7D4A7E/officedeploymenttool_18129-20030.exe" // 10MB
	DefaultFilter        = "all"
	DefaultPingMethod    = "http"  // "http", "tcp"
	DefaultSortMethod    = "speed" // "speed", "ping"

	// 性能配置默认值
	DefaultConcurrency   = 5
	DefaultTimeout       = 10 * time.Second
	DefaultBufferSize    = 32 * 1024 // 32KB
	DefaultRetryAttempts = 2
	DefaultRetryDelay    = 1 * time.Second

	// 日志配置默认值
	DefaultLogLevel    = "info"
	DefaultLogFile     = "speedtest.log"
	DefaultGeoIPDBPath = "GeoLite2-Country.mmdb"

	// 下载配置默认值
	DefaultDownloadTimeout    = 30 * time.Second
	DefaultDownloadRetries    = 2
	DefaultDownloadBufferSize = 32 * 1024 // 32KB
)

// Config 全局配置
type Config struct {
	// 基本配置
	GroupName     string `json:"group"`
	SpeedTestMode string `json:"speedtestMode"`
	PingURL       string `json:"pingUrl"`
	DownloadURL   string `json:"downloadUrl"`
	Filter        string `json:"filter"`
	PingMethod    string `json:"pingMethod"`
	SortMethod    string `json:"sortMethod"`

	// 性能配置
	Concurrency   int           `json:"concurrency"`
	Timeout       time.Duration `json:"timeout"`
	BufferSize    int           `json:"bufferSize"`
	RetryAttempts int           `json:"retryAttempts"`
	RetryDelay    time.Duration `json:"retryDelay"`

	// 功能开关
	Detect        bool `json:"detect"`
	RemoveDup     bool `json:"removeDup"`
	EnableMetrics bool `json:"enableMetrics"`
	RemoteIP      bool `json:"remoteIP"`

	// 日志配置
	LogLevel    string `json:"logLevel"`
	LogFile     string `json:"logFile"`
	GeoIPDBPath string `json:"geoipDbPath"`

	// 下载配置
	DownloadTimeout    time.Duration `json:"download_timeout"`
	DownloadRetries    int           `json:"download_retries"`
	DownloadBufferSize int           `json:"download_buffer_size"`

	// 内部状态跟踪
	Initialized bool `json:"-"` // 使用 json:"-" 在序列化时忽略此字段
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	config := &Config{
		SpeedTestMode: AllTest,
		PingURL:       DefaultPingURL,
		DownloadURL:   DefaultDownloadURL,
		Concurrency:   DefaultConcurrency,
		Timeout:       DefaultTimeout,
		BufferSize:    DefaultBufferSize,
		RetryAttempts: DefaultRetries,
		RetryDelay:    time.Millisecond * 100,
		EnableMetrics: true,
		LogLevel:      "info",
		Initialized:   true, // 标记为已初始化
	}
	return config
}

// InitWithDefaults 使用默认值初始化配置
func (c *Config) InitWithDefaults() {
	fmt.Println("Init_values: ", c.Initialized)
	// 如果已经初始化过，直接返回
	if c.Initialized {
		return
	}

	// 基本配置初始化
	if c.GroupName == "" {
		c.GroupName = DefaultGroupName
	}
	if c.SpeedTestMode == "" {
		c.SpeedTestMode = DefaultSpeedTestMode
	}
	if c.PingURL == "" {
		c.PingURL = DefaultPingURL
	}
	if c.DownloadURL == "" {
		c.DownloadURL = DefaultDownloadURL
	}
	if c.Filter == "" {
		c.Filter = DefaultFilter
	}
	if c.PingMethod == "" {
		c.PingMethod = DefaultPingMethod
	}
	if c.SortMethod == "" {
		c.SortMethod = DefaultSortMethod
	}

	// 性能配置初始化
	if c.Concurrency == 0 {
		c.Concurrency = DefaultConcurrency
	}
	// 修改后的代码
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	} else {
		// 将输入值视为秒数，转换为 Duration
		timeoutSeconds := c.Timeout.Seconds()
		fmt.Printf("c.Timeout: %v, timeoutSeconds: %v\n", c.Timeout, timeoutSeconds)
		if timeoutSeconds > 16 {
			c.Timeout = 16 * time.Second
		} else if timeoutSeconds > 0 && timeoutSeconds < 16 {
			c.Timeout = time.Duration(timeoutSeconds) * time.Second
		}
	}

	if c.BufferSize == 0 {
		c.BufferSize = DefaultBufferSize
	}
	if c.RetryAttempts == 0 {
		c.RetryAttempts = DefaultRetryAttempts
	}
	if c.RetryDelay == 0 {
		c.RetryDelay = DefaultRetryDelay
	}

	// 日志配置初始化
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	if c.LogFile == "" {
		c.LogFile = DefaultLogFile
	}
	if c.GeoIPDBPath == "" {
		c.GeoIPDBPath = DefaultGeoIPDBPath
	}

	// 下载配置初始化
	if c.DownloadTimeout == 0 {
		c.DownloadTimeout = DefaultDownloadTimeout
	} else if c.DownloadTimeout < time.Second {
		c.DownloadTimeout = time.Duration(c.DownloadTimeout) * time.Second
	} else {
		c.DownloadTimeout = DefaultDownloadTimeout
	}
	if c.DownloadRetries == 0 {
		c.DownloadRetries = DefaultDownloadRetries
	}
	if c.DownloadBufferSize == 0 {
		c.DownloadBufferSize = DefaultDownloadBufferSize
	}

	// 标记为已初始化
	c.Initialized = true
	// fmt.Println("Parsed config: ", c)
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	// 验证基本配置
	if c.SpeedTestMode != "all" && c.SpeedTestMode != "pingonly" && c.SpeedTestMode != "speedonly" {
		return fmt.Errorf("invalid speedtest_mode: %s", c.SpeedTestMode)
	}
	if c.PingMethod != "http" && c.PingMethod != "tcp" {
		return fmt.Errorf("invalid ping_method: %s", c.PingMethod)
	}
	if c.SortMethod != "speed" && c.SortMethod != "ping" {
		return fmt.Errorf("invalid sort_method: %s", c.SortMethod)
	}

	// 验证性能配置
	if c.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second")
	}
	if c.BufferSize < 1024 {
		return fmt.Errorf("buffer_size must be at least 1KB")
	}
	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts cannot be negative")
	}

	// 验证下载配置
	if c.DownloadTimeout < time.Second {
		fmt.Println(c.DownloadTimeout)
		return fmt.Errorf("download_timeout must be at least 1 second")
	}
	if c.DownloadRetries < 0 {
		return fmt.Errorf("download_retries cannot be negative")
	}
	if c.DownloadBufferSize < 1024 {
		return fmt.Errorf("download_buffer_size must be at least 1KB")
	}

	// 验证地理位置检测相关配置
	if c.Detect && c.GeoIPDBPath == "" {
		return fmt.Errorf("geoip_db_path must be set when detect is enabled")
	}

	return nil
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{\n"+
			"  GroupName: %s\n"+
			"  SpeedTestMode: %s\n"+
			"  PingURL: %s\n"+
			"  DownloadURL: %s\n"+
			"  Filter: %s\n"+
			"  PingMethod: %s\n"+
			"  SortMethod: %s\n"+
			"  Concurrency: %d\n"+
			"  Timeout: %v\n"+
			"  BufferSize: %d\n"+
			"  RetryAttempts: %d\n"+
			"  RetryDelay: %v\n"+
			"  Detect: %v\n"+
			"  RemoveDup: %v\n"+
			"  EnableMetrics: %v\n"+
			"  RemoteIP: %v\n"+
			"  LogLevel: %s\n"+
			"  LogFile: %s\n"+
			"  GeoIPDBPath: %s\n"+
			"  DownloadTimeout: %v\n"+
			"  DownloadRetries: %d\n"+
			"  DownloadBufferSize: %d\n"+
			"}",
		c.GroupName,
		c.SpeedTestMode,
		c.PingURL,
		c.DownloadURL,
		c.Filter,
		c.PingMethod,
		c.SortMethod,
		c.Concurrency,
		c.Timeout,
		c.BufferSize,
		c.RetryAttempts,
		c.RetryDelay,
		c.Detect,
		c.RemoveDup,
		c.EnableMetrics,
		c.RemoteIP,
		c.LogLevel,
		c.LogFile,
		c.GeoIPDBPath,
		c.DownloadTimeout,
		c.DownloadRetries,
		c.DownloadBufferSize,
	)
}
