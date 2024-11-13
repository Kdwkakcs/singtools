# SingTools 使用说明

SingTools 是一个基于 sing-box 的节点测试工具，可以测试节点的延迟和速度。支持多种协议，包括 Shadowsocks、Trojan、VMess、Hysteria、Hysteria2、Vless 等。

## 功能特点

   - 支持多种协议测试
   - 并发测试提高效率
   - 支持延迟和速度测试
   - 支持地理位置检测
   - 支持节点去重和排序
   - 提供详细的测试统计

## 安装说明
### 从 Release 下载

访问 [GitHub Releases](https://github.com/Kdwkakcs/singtools/releases) 页面下载对应平台的预编译版本：

## 1. 命令概述

```bash
Usage:
  singtools test [flags]

Flags:
  -c, --config string   Path to test configuration file that defines test parameters:
                        - Test URLs for connectivity checks
                        - Timeout settings for various operations
                        - Speed test parameters and thresholds
                        - Other test-specific configurations
  -d, --detect          Enable country detection for proxies.
                        When enabled:
                        - Determines the geographical location of each node
                        - Adds country information to the test results
                        - Helps in organizing nodes by region
                        Note: Enabling this option may increase overall processing time
  -f, --filter string   Filter nodes based on protocol type.
                        Specify comma-separated list of protocols to include.
                        Supported protocols:
                        - shadowsocks
                        - vmess
                        - trojan
                        - tls
                        Default: 'tls, shadowsocks' (default "tls, shadowsocks")
  -h, --help            help for test
  -i, --input string    Path to input configuration file. Supports:
                        - Local file path (e.g., ./config.json)
                        - HTTP/HTTPS URL for remote config
                        - Sing-box format configuration
                        This parameter is required.
  -e, --level string    Set logging level for operation feedback.
                        Available options:
                        - debug: Show detailed debugging information
                        - info:  Show informational messages
                        - warn:  Show warning messages (default)
                        - error: Show only error messages
                        - fatal: Show only fatal errors
                        Higher levels include all lower level messages (default "warn")
  -m, --meta string     Path to save detailed test results metadata including:
                        - Comprehensive latency measurements
                        - Detailed speed test results
                        - Connection success/failure statistics
                        - Country information (if detection is enabled)
                        - Protocol-specific performance data
                        Default: meta.json in current directory (default "meta.json")
  -o, --output string   Path to save the filtered and tested configuration file.
                        The output file will contain only working nodes that passed the tests.
                        Default: out.json in current directory (default "out.json")
  -r, --remote          Enable remote IP detection for each proxy.
                        When enabled:
                        - Tests and records the exit IP for each node
                        - Verifies actual proxy endpoint
                        - Helps identify unique proxies
                        Note: This will increase testing time per node

Global Flags:
  -v, --version   Show version information
```

主要用于测试 sing-box 支持的所有协议的延迟和速度，支持本地配置文件和在线订阅链接，同时支持多种配置文件格式（包括 mixed, clash, sing-box等），目前仅支持输出为sing-box格式，可以使用subconverter(https://github.com/asdlokj1qpi233/subconverter)转换为其他格式。

### 简短说明如何使用subconverter进行不同订阅格式之间的转换，后期可能会增加输出为其他格式的功能
```bash
# 安装subconverter
# 从 https://github.com/asdlokj1qpi233/subconverter/releases release页面下载对应平台的预编译版本，解压后运行即可。
# 其他的版本的subconverter暂时不支持singbox格式转换为其其他格式，
./subconverter 

# 将singtools的输出结果转换为clash格式，/path/to/singtools/out.json为你本地singtools的输出结果路径，windows中自行尝试路径以及如何调用subconverter，可能需要将\\替换为/
curl http://127.0.0.1:25500/sub?target=clash&url=/path/to/singtools/out.json
# 或者在本地游览器中打开 http://127.0.0.1:25500/sub?target=clash&url=/path/to/singtools/out.json
```

## 2. 命令参数

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| --input | -i | 无(必填) | 输入文件路径，支持本地文件或 HTTP(S) URL |
| --output | -o | out.json | 输出文件路径 |
| --config | -c | 无 | 配置文件路径 |
| --detect | -d | false | 是否检测节点所在国家 |
| --meta | -m | meta.json | 元数据输出文件路径 |
| --download | -l | false | 是否下载 MMDB 数据库 |
| --level | -e | warn | 日志级别(trace/debug/info/warn/error/fatal/panic) |
| --remote | -r | false | 是否获取远程 IP |
| --filter | -f | "tls, shadowsocks" | 基于类型过滤节点，用逗号分隔 |

## 3. 配置文件说明

配置文件使用 JSON 格式，支持以下主要配置项：

### 3.1 基本配置

```json
{
  "group": "Default",
  "speedtestMode": "all",
  "pingUrl": "https://www.gstatic.com/generate_204",
  "downloadUrl": "https://download.microsoft.com/download/2/7/A/27AF1BE6-DD20-4CB4-B154-EBAB8A7D4A7E/officedeploymenttool_18129-20030.exe",
  "filter": "all",
  "pingMethod": "http",
  "sortMethod": "speed"
}
```

### 3.2 性能配置

```json
{
  "concurrency": 5,
  "timeout": 10,
  "bufferSize": 32768,
  "retryAttempts": 2,
  "retryDelay": 1
}
```

### 3.3 功能开关

```json
{
  "detect": false,
  "removeDup": false,
  "enableMetrics": true,
  "remoteIP": false
}
```

## 4. 主要功能

1. **延迟测试**：
   - 支持 HTTP 和 TCP 两种测试方式
   - 默认使用 Google 生成的 204 页面测试
   - 可配置重试次数和超时时间

2. **速度测试**：
   - 支持并发测试
   - 可配置下载文件大小和缓冲区大小
   - 提供平均速度和最大速度数据

3. **地理位置检测**：
   - 支持自动下载 GeoLite2 数据库
   - 可检测节点所在国家/地区
   - 支持 IP 地址缓存

4. **结果处理**：
   - 支持节点去重
   - 支持多种排序方式（速度/延迟）
   - 提供详细的测试统计信息

## 5. 使用示例

### 5.1 基本使用

```bash
# 测试本地配置文件
singtools test -i sing-box.json

# 测试在线订阅链接
singtools test -i https://example.com/sub

# 指定输出文件
singtools test -i sing-box.json -o result.json
```

### 5.2 高级使用

```bash
# 使用自定义配置文件测试
singtools test -i sing-box.json -c custom_config.json

# 启用国家检测并下载 MMDB
singtools test -i sing-box.json -d -l

# 使用详细日志级别
singtools test -i sing-box.json -e debug

# 过滤特定类型节点（使用逗号分隔），默认过滤tls和shadowsocks，如果不想过滤，则使用 -f ""
singtools test -i sing-box.json -f "vmess,trojan"
```

## 6. 注意事项

1. 输入文件必须是有效的 sing-box 配置格式，如果不是会自动尝试转换
2. 超时设置：
   - 单个节点测试超时默认为 10 秒
   - 下载测试超时默认为 30 秒
3. 并发设置：
   - 默认并发数为 5
   - 可通过配置文件调整

## 7. 输出说明

工具会生成两个主要输出文件：

1. **测试结果文件** (默认: out.json)：
   - 包含所有可用节点的完整配置
   - 按照指定方式排序

2. **元数据文件** (默认: meta.json)：
   - 包含节点的测试数据
   - 包括延迟、速度、地理位置等信息
