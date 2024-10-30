package ping

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dkwkoaca/singtools/utils"

	"github.com/Dkwkoaca/singtools/log"
	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

// TestManager 测试管理器
type TestManager struct {
	config     *Config
	resources  *ResourcePool
	metrics    *Metrics
	logger     *log.Logger
	geoManager *GeoIPManager
	ctx        context.Context
	cancel     context.CancelFunc
	workerPool *WorkerPool
}

// NewTestManager 创建测试管理器
func NewTestManager(config *Config) (*TestManager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建 logger
	logger := log.NewLogger(config.LogLevel)

	// 创建资源池
	resources := NewResourcePool(config)

	// 创建指标收集器
	metrics := NewMetrics()

	// 创建工作池
	workerPool := NewWorkerPool(config.Concurrency)

	// 创建 GeoIPManager
	geoManager, err := NewGeoIPManager(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create GeoIPManager failed: %w", err)
	}

	manager := &TestManager{
		config:     config,
		resources:  resources,
		metrics:    metrics,
		logger:     logger, // 确保设置 logger
		geoManager: geoManager,
		workerPool: workerPool,
		ctx:        ctx,
		cancel:     cancel,
	}

	return manager, nil
}

// Close 关闭管理器
func (m *TestManager) Close() error {
	m.cancel()
	if m.geoManager != nil {
		return m.geoManager.Close()
	}
	return nil
}

// ResourcePool 资源池
type ResourcePool struct {
	bufferPool *sync.Pool
	clientPool *sync.Pool
}

func NewResourcePool(config *Config) *ResourcePool {
	return &ResourcePool{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, config.BufferSize)
			},
		},
		clientPool: &sync.Pool{
			New: func() interface{} {
				return &http.Client{
					Timeout: config.Timeout,
					Transport: &http.Transport{
						MaxIdleConns:       100,
						IdleConnTimeout:    90 * time.Second,
						DisableCompression: true,
						DisableKeepAlives:  false,
					},
				}
			},
		},
	}
}

// Metrics 测试指标
type Metrics struct {
	testCount    atomic.Int64
	failureCount atomic.Int64
	lastTestTime atomic.Value // stores time.Time
}

// Stats 统计信息
type Stats struct {
	TotalNodes   int64
	SuccessNodes int64
	FailedNodes  int64
	SuccessRate  float64
	LastTestTime time.Time
}

// NewMetrics 创建指标收集器
func NewMetrics() *Metrics {
	m := &Metrics{}
	m.lastTestTime.Store(time.Now())
	return m
}

// GetStats 获取统计信息
func (m *Metrics) GetStats() Stats {
	total := m.testCount.Load()
	failed := m.failureCount.Load()
	success := total - failed

	var successRate float64
	if total > 0 { // 添加除零检查
		successRate = float64(success) / float64(total) * 100
	}

	return Stats{
		TotalNodes:   total,
		SuccessNodes: success,
		FailedNodes:  failed,
		SuccessRate:  successRate,
		LastTestTime: m.lastTestTime.Load().(time.Time),
	}
}

// URLTest 执行URL测试
func (m *TestManager) URLTest(ctx context.Context, out adapter.Outbound) (Node, error) {
	// 获取ping URL
	link := m.config.PingURL
	if link == "" {
		link = "https://www.gstatic.com/generate_204"
	}

	// 解析URL
	linkURL, err := url.Parse(link)
	if err != nil {
		m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v", out.Tag(), out.Type(), err))
		return CreateBlockNode(out), err
	}

	// 获取主机名和端口
	hostname := linkURL.Hostname()
	port := linkURL.Port()
	if port == "" {
		switch linkURL.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	// 开始计时
	start := time.Now()

	// 建立连接
	instance, err := out.DialContext(ctx, "tcp", M.ParseSocksaddrHostPortStr(hostname, port))
	if err != nil {
		// m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v", out.Tag(), out.Type(), err))
		return CreateBlockNode(out), err
	}
	defer instance.Close()

	// remoteAddr, ok := instance.RemoteAddr().(*net.TCPAddr)
	// if !ok {
	// 	m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v, remoteAddr: %v", out.Tag(), out.Type(), err, instance.RemoteAddr()))
	// }
	instance.SetDeadline(time.Now().Add(m.config.Timeout))
	instance.SetReadDeadline(time.Now().Add(m.config.Timeout))

	// 创建HTTP请求
	req, err := http.NewRequest(http.MethodHead, link, nil)
	if err != nil {
		// m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v", out.Tag(), out.Type(), err))
		return CreateBlockNode(out), err
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return instance, nil
		},
		DisableKeepAlives: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   m.config.Timeout,
	}
	defer func() {
		transport.CloseIdleConnections()
		client.CloseIdleConnections()
	}()

	// 执行请求
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v", out.Tag(), out.Type(), err))
		return CreateBlockNode(out), err
	}
	defer resp.Body.Close()

	// 计算延迟
	delay := time.Since(start).Milliseconds()

	// 使用构建器创建节点
	builder := NewNodeBuilder(out).WithPing(delay)

	// 如果需要检测国家
	if m.config.Detect && m.geoManager != nil {
		if remoteAddr, ok := instance.RemoteAddr().(*net.TCPAddr); ok {
			ip := remoteAddr.IP.String()
			if country, err := m.geoManager.GetCountry(ip); err == nil {
				builder.WithIP(ip, country)

				// 如果需要获取远程IP
				if m.config.RemoteIP {
					if remoteIPs, remoteCountries, err := utils.GetIP(out); err == nil {
						builder.WithRemoteIP(remoteIPs, remoteCountries)
					}
				}
			}
		} else {
			builder.WithIP("", "")
			m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v, remoteAddr: %v", out.Tag(), out.Type(), err, instance.RemoteAddr()))
		}
	}

	return builder.Build(), nil
}

// getTestTimeout 根据测试模式返回适当的超时时间
func (m *TestManager) getTestTimeout(mode string) time.Duration {
	switch mode {
	case SpeedOnly, AllTest:
		return m.config.DownloadTimeout
	default:
		return m.config.Timeout
	}
}

func (m *TestManager) TestAll(ctx context.Context, outbounds []adapter.Outbound) ([]Node, error) {
	outLen := len(outbounds)
	if outLen < 1 {
		return nil, ErrNoOutbound
	}

	m.logger.Debug(fmt.Sprintf("Starting TestAll with %d outbounds in %s mode", outLen, m.config.SpeedTestMode))
	startTime := time.Now()

	var (
		eg     errgroup.Group
		mutex  sync.Mutex
		nodes  = make([]Node, 0, outLen)
		sem    = make(chan struct{}, m.config.Concurrency)
		tested atomic.Int64
	)

	// 创建进度条，显示总节点数但按百分比更新
	description := fmt.Sprintf("Testing %d nodes", outLen)
	bar := progressbar.NewOptions(100,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowElapsedTimeOnFinish(),
	)

	// 启动进度更新协程
	done := make(chan struct{})
	go func() {
		defer close(done)
		lastPercent := -1 // 初始化为-1确保第一次更新
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				completed := tested.Load()
				currentPercent := int(float64(completed) * 100 / float64(outLen))

				// 只在百分比增加1%时更新
				if currentPercent > lastPercent {
					_ = bar.Set(currentPercent)
					lastPercent = currentPercent

					// 更新描述以显示当前进度
					description := fmt.Sprintf("Testing %d/%d nodes (%d%%)",
						completed, outLen, currentPercent)
					bar.Describe(description)
				}

				if completed >= int64(outLen) {
					_ = bar.Finish()
					return
				}
			}
		}
	}()

	// 其余测试逻辑保持不变
	timeout := m.getTestTimeout(m.config.SpeedTestMode)

	for _, out := range outbounds {
		out := out
		eg.Go(func() error {
			select {
			case sem <- struct{}{}:
				defer func() {
					<-sem
					tested.Add(1)
				}()
			case <-ctx.Done():
				return ctx.Err()
			}

			nodeCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			node, err := m.TestNode(nodeCtx, out)
			if err != nil {
				m.logger.Debug(fmt.Sprintf("Test failed for %s(%s): %v", out.Tag(), out.Type(), err))
				return nil
			}

			mutex.Lock()
			nodes = append(nodes, node)
			mutex.Unlock()

			return nil
		})
	}

	// 等待所有测试完成
	err := eg.Wait()

	// 等待进度条更新完成
	<-done

	if err != nil {
		return nil, fmt.Errorf("test all nodes failed: %w", err)
	}

	m.logger.Debug(fmt.Sprintf("TestAll completed in %s", time.Since(startTime)))
	return nodes, nil
}

// TODO 根据测试的内容修改ctx的timeout
func (m *TestManager) TestNode(ctx context.Context, out adapter.Outbound) (Node, error) {
	// 创建一个默认的阻塞节点
	blockNode := CreateBlockNode(out)

	defer func() {
		m.metrics.testCount.Add(1)
		m.metrics.lastTestTime.Store(time.Now())
	}()

	// 添加 select 来处理上下文取消
	select {
	case <-ctx.Done():
		// 上下文已取消，发送阻塞节点并返回错误
		m.metrics.failureCount.Add(1)
		return blockNode, ctx.Err()
	default:
		node, err := m.processNode(ctx, out)
		if err != nil {
			m.metrics.failureCount.Add(1)
			return blockNode, err
		}
		return node, nil
	}
}

// processNode 处理单个节点的测试
func (m *TestManager) processNode(ctx context.Context, out adapter.Outbound) (Node, error) {
	switch m.config.SpeedTestMode {
	case PingOnly:
		return m.URLTest(ctx, out)
	case SpeedOnly:
		return m.SpeedTest(ctx, out)
	case AllTest:
		node, err := m.URLTest(ctx, out)
		if err != nil {
			return CreateBlockNode(out), err
		}

		if !node.IsOk {
			return node, nil
		}

		// 执行速度测试
		if err := m.runSpeedTest(ctx, &node, out); err != nil {
			m.logger.Debug(fmt.Sprintf("Speed test failed for %s: %v", out.Tag(), err))
			// 即使速度测试失败，也返回带有ping结果的节点
			return node, nil
		}

		return node, nil
	default:
		return CreateBlockNode(out), fmt.Errorf("unknown test mode: %s", m.config.SpeedTestMode)
	}
}

// SpeedTest 执行速度测试
func (m *TestManager) SpeedTest(ctx context.Context, out adapter.Outbound) (Node, error) {
	node := CreateBlockNode(out)

	buffer := m.resources.bufferPool.Get().([]byte)
	defer m.resources.bufferPool.Put(buffer)

	// downloader := NewDownloadTester(m.config, m.resources, m.metrics)
	if err := m.runSpeedTest(ctx, &node, out); err != nil {
		return node, fmt.Errorf("speed test failed: %w", err)
	}

	node.IsOk = true
	return node, nil
}
