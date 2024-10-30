package ping

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
)

// // TestNode 测试单个节点
// func (m *TestManager) TestNode(ctx context.Context, out adapter.Outbound) (Node, error) {
// 	// 创建一个带超时的上下文
// 	testCtx, cancel := context.WithTimeout(ctx, m.config.Timeout)
// 	defer cancel()

// 	// 更新指标
// 	defer func() {
// 		m.metrics.lastTestTime.Store(time.Now())
// 		m.metrics.testCount.Add(1)
// 	}()

// 	// 执行 ping 测试
// 	node, err := m.URLTest(testCtx, out)
// 	if err != nil {
// 		m.metrics.failureCount.Add(1)
// 		return NewBlockNode(out), err
// 	}

// 	// 在 PingOnly 模式下直接返回结果
// 	if m.config.SpeedTestMode == PingOnly {
// 		return node, nil
// 	}

// 	// 如果节点不可用，直接返回
// 	if !node.IsOk {
// 		return node, nil
// 	}

// 	// 执行速度测试
// 	if err := m.runSpeedTest(testCtx, &node, out); err != nil {
// 		m.logger.Debug(fmt.Sprintf("Speed test failed for %s: %v", out.Tag(), err))
// 		// 即使速度测试失败，也返回带有ping结果的节点
// 		return node, nil
// 	}

// 	return node, nil
// }

// runSpeedTest 执行速度测试并更新节点信息
func (m *TestManager) runSpeedTest(ctx context.Context, node *Node, out adapter.Outbound) error {
	speedChan := make(chan int64, 100)
	startChan := make(chan time.Time, 1)
	doneChan := make(chan struct{})
	defer close(speedChan)
	defer close(startChan)

	// 创建统计结构
	var stats struct {
		sync.Mutex
		max   int64
		sum   int64
		count int64
		start time.Time
	}

	// 启动统计协程
	go func() {
		defer close(doneChan)
		for {
			select {
			case speed, ok := <-speedChan:
				if !ok {
					return
				}
				if speed < 0 {
					continue
				}
				stats.Lock()
				stats.sum += speed
				stats.count++
				if stats.max < speed {
					stats.max = speed
				}
				stats.Unlock()
			case start := <-startChan:
				stats.Lock()
				stats.start = start
				stats.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	// 获取缓冲区
	buffer := m.resources.bufferPool.Get().([]byte)
	defer m.resources.bufferPool.Put(buffer)

	// 执行下载测试
	downloader := NewDownloadTester(m.config, m.resources, m.metrics)
	testErr := downloader.download(ctx, out, buffer, speedChan, startChan)

	// 等待统计完成
	select {
	case <-doneChan:
	case <-time.After(3 * time.Second):
		m.logger.Debug(fmt.Sprintf("Warning: Statistics collection timed out for %s", out.Tag()))
	}

	// 更新节点统计信息
	stats.Lock()
	defer stats.Unlock()

	if stats.count > 0 {
		duration := time.Since(stats.start).Seconds()
		if duration > 0 {
			node.AvgSpeed = int64(float64(stats.sum) / duration)
			if node.AvgSpeed < 0 {
				node.AvgSpeed = 0
			}
		}
		node.MaxSpeed = stats.max
		if node.MaxSpeed < 0 {
			node.MaxSpeed = 0
		}
	}

	return testErr
}

// doPingTest 执行ping测试
func (m *TestManager) doPingTest(ctx context.Context, out adapter.Outbound) (Node, error) {
	start := time.Now()

	link := "http://www.gstatic.com/generate_204"
	linkURL, err := url.Parse(link)
	hostname := linkURL.Hostname()
	conn, err := out.DialContext(ctx, "tcp", M.ParseSocksaddrHostPort(hostname, 80))
	if err != nil {
		return CreateBlockNode(out), err
	}
	defer conn.Close()

	// 设置超时
	deadline := time.Now().Add(m.config.Timeout)
	conn.SetDeadline(deadline)

	// 执行HTTP请求
	req, err := http.NewRequestWithContext(ctx, "HEAD", m.config.PingURL, nil)
	if err != nil {
		return CreateBlockNode(out), err
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return conn, nil
			},
		},
	}
	defer client.CloseIdleConnections()

	resp, err := client.Do(req)
	if err != nil {
		return CreateBlockNode(out), err
	}
	defer resp.Body.Close()

	latency := time.Since(start).Milliseconds()

	// 创建节点信息
	node := CreateNodeWithPing(out, latency)

	// 如果需要获取地理位置信息
	if m.config.Detect {
		if m.geoManager != nil {
			ip := conn.RemoteAddr().String()
			country, err := m.geoManager.GetCountry(ip)
			if err == nil {
				node.Ip = ip
				node.Country = country
			}
		}
	}

	return node, nil
}

// doSpeedTest 执行速度测试
func (m *TestManager) doSpeedTest(ctx context.Context, node *Node, out adapter.Outbound, buffer []byte) error {
	speedChan := make(chan int64, 100)
	startChan := make(chan time.Time, 1)
	defer close(speedChan)
	defer close(startChan)

	var stats struct {
		sync.Mutex
		max   int64
		sum   int64
		count int
	}

	// 启动速度统计
	go func() {
		for speed := range speedChan {
			stats.Lock()
			stats.sum += speed
			stats.count++
			if stats.max < speed {
				stats.max = speed
			}
			stats.Unlock()
		}
	}()

	// 执行下载测试
	downloader := NewDownloadTester(m.config, m.resources, m.metrics)
	if err := downloader.download(ctx, out, buffer, speedChan, startChan); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// 更新节点统计信息
	stats.Lock()
	if stats.count > 0 {
		node.AvgSpeed = stats.sum / int64(stats.count)
		node.MaxSpeed = stats.max
	}
	stats.Unlock()

	return nil
}
