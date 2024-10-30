package ping

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/sagernet/sing-box/adapter"
	// "github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
)

// const DefaultUserAgent = "SpeedTest/1.0"

// DownloadTester 下载测试器
type DownloadTester struct {
	config    *Config
	resources *ResourcePool
	metrics   *Metrics
}

// NewDownloadTester 创建下载测试器
func NewDownloadTester(config *Config, resources *ResourcePool, metrics *Metrics) *DownloadTester {
	return &DownloadTester{
		config:    config,
		resources: resources,
		metrics:   metrics,
	}
}

// download 执行下载测试
func (d *DownloadTester) download(ctx context.Context, out adapter.Outbound, buffer []byte, speedChan chan<- int64, startChan chan<- time.Time) error {
	// 创建一个子上下文，用于控制超时
	downloadCtx, cancel := context.WithTimeout(ctx, d.config.DownloadTimeout)
	defer cancel()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return out.DialContext(ctx, network, M.ParseSocksaddr(addr))
		},
		ForceAttemptHTTP2: true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   d.config.DownloadTimeout,
	}
	// defer client.CloseIdleConnections()
	urlParsed, err := url.Parse(d.config.DownloadURL)
	request, err := http.NewRequest("GET", urlParsed.String(), nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	request.Header.Set("User-Agent", DefaultUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer response.Body.Close()

	// if response.StatusCode != http.StatusOK {
	// 	return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	// }

	// 发送开始时间
	startTime := time.Now()
	select {
	case startChan <- startTime:
	default:
	}

	var (
		total int64
		prev  = startTime
	)

	for {
		select {
		case <-downloadCtx.Done():
			// 确保发送最后的数据
			if total > 0 {
				select {
				case speedChan <- total:
				default:
				}
			}
			return downloadCtx.Err()

		default:
			nr, er := response.Body.Read(buffer)
			if nr > 0 {
				total += int64(nr)
			}

			now := time.Now()
			if now.Sub(prev) >= time.Second || er != nil {
				if total > 0 {
					speed := total
					if speed < 0 {
						speed = 0
					}
					select {
					case speedChan <- speed:
					default:
					}
					total = 0
				}
				prev = now
			}

			if er != nil {
				// 确保发送最后的数据
				if total > 0 {
					select {
					case speedChan <- total:
					default:
					}
				}
				if er == io.EOF {
					return nil
				}
				return fmt.Errorf("read response failed: %w", er)
			}
		}
	}
}

func (d *DownloadTester) doDownload(ctx context.Context, client *http.Client, buffer []byte, speedChan chan<- int64, startChan chan<- time.Time) error {
	req, err := http.NewRequestWithContext(ctx, "GET", d.config.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	// 发送开始时间
	startTime := time.Now()
	select {
	case startChan <- startTime:
	default:
		// 如果channel已满，继续执行
	}

	var (
		total int64
		max   int64
		prev  = startTime
	)

	for {
		select {
		case <-ctx.Done():
			// 发送最后一次速度数据
			if total > 0 {
				select {
				case speedChan <- total:
				default:
				}
			}
			return ctx.Err()

		default:
			nr, er := response.Body.Read(buffer)
			if nr > 0 {
				total += int64(nr)
			}

			now := time.Now()
			if now.Sub(prev) >= time.Second || er != nil {
				if total > 0 {
					select {
					case speedChan <- total:
						if max < total {
							max = total
						}
						total = 0
					default:
					}
				}
				prev = now
			}

			if er != nil {
				if er == io.EOF {
					return nil
				}
				// 如果是超时错误，确保发送最后的数据
				if er == context.DeadlineExceeded {
					if total > 0 {
						select {
						case speedChan <- total:
						default:
						}
					}
				}
				return fmt.Errorf("read response failed: %w", er)
			}
		}
	}
}

// processDownload 处理下载流程
func (d *DownloadTester) processDownload(ctx context.Context, body io.ReadCloser, buffer []byte, speedChan chan<- int64) error {
	var total int64
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// 使用标准库的 bufio.NewReaderSize
	reader := bufio.NewReaderSize(body, 32*1024) // 32KB 缓冲

	// 设置读取超时
	deadline := time.Now().Add(15 * time.Second)
	if deadline, ok := ctx.Deadline(); ok {
		deadline = deadline
	}

	done := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return
			case <-ticker.C:
				if total > 0 {
					select {
					case speedChan <- total:
						total = 0
					default:
					}
				}
			default:
				if time.Now().After(deadline) {
					done <- fmt.Errorf("read timeout")
					return
				}

				n, err := reader.Read(buffer)
				if n > 0 {
					total += int64(n)
				}
				if err != nil {
					if err == io.EOF {
						done <- nil
						return
					}
					done <- fmt.Errorf("read response failed: %w", err)
					return
				}
			}
		}
	}()

	return <-done
}
