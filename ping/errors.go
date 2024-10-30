package ping

import (
	"errors"
	"fmt"
)

var (
	// ErrNoOutbound 没有可用的出站连接
	ErrNoOutbound = errors.New("no outbound available")

	// ErrTimeout 操作超时
	ErrTimeout = errors.New("operation timeout")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrMaxNodesReached 达到最大节点数
	ErrMaxNodesReached = errors.New("maximum nodes reached")

	// ErrTestFailed 测试失败
	ErrTestFailed = errors.New("test failed")

	// ErrSpeedTestFailed 速度测试失败
	ErrSpeedTestFailed = errors.New("speed test failed")
)

// PingError ping测试错误
type PingError struct {
	Outbound string
	Err      error
}

func (e *PingError) Error() string {
	return fmt.Sprintf("ping error for %s: %v", e.Outbound, e.Err)
}

// SpeedTestError 速度测试错误
type SpeedTestError struct {
	Outbound string
	Err      error
}

func (e *SpeedTestError) Error() string {
	return fmt.Sprintf("speed test error for %s: %v", e.Outbound, e.Err)
}

// NewPingError 创建ping错误
func NewPingError(outbound string, err error) error {
	return &PingError{
		Outbound: outbound,
		Err:      err,
	}
}

// NewSpeedTestError 创建速度测试错误
func NewSpeedTestError(outbound string, err error) error {
	return &SpeedTestError{
		Outbound: outbound,
		Err:      err,
	}
}
