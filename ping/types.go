package ping

import (
	"context"
	"time"

	"github.com/sagernet/sing-box/adapter"
)

// Node 节点信息
type Node struct {
	Tag           string    `json:"id,omitempty"`
	Protocol      string    `json:"protocol,omitempty"`
	Ping          int64     `json:"ping,omitempty"`
	AvgSpeed      int64     `json:"avg_speed,omitempty"`
	MaxSpeed      int64     `json:"max_speed,omitempty"`
	Ip            string    `json:"ip,omitempty"`
	Country       string    `json:"country,omitempty"`
	RemoteIP      string    `json:"remote_ip,omitempty"`
	RemoteCountry string    `json:"remote_country,omitempty"`
	IsOk          bool      `json:"isok,omitempty"`
	TestTime      time.Time `json:"test_time,omitempty"`
}

// NodeBuilder Node 构建器
type NodeBuilder struct {
	node Node
}

// NewNodeBuilder 创建节点构建器
func NewNodeBuilder(out adapter.Outbound) *NodeBuilder {
	return &NodeBuilder{
		node: Node{
			Tag:      out.Tag(),
			Protocol: out.Type(),
			TestTime: time.Now(),
		},
	}
}

// WithPing 设置延迟
func (b *NodeBuilder) WithPing(ping int64) *NodeBuilder {
	b.node.Ping = ping
	b.node.IsOk = true
	return b
}

// WithIP 设置 IP 信息
func (b *NodeBuilder) WithIP(ip, country string) *NodeBuilder {
	b.node.Ip = ip
	b.node.Country = country
	return b
}

// WithRemoteIP 设置远程 IP 信息
func (b *NodeBuilder) WithRemoteIP(ip, country string) *NodeBuilder {
	b.node.RemoteIP = ip
	b.node.RemoteCountry = country
	return b
}

// WithSpeed 设置速度信息
func (b *NodeBuilder) WithSpeed(avg, max int64) *NodeBuilder {
	b.node.AvgSpeed = avg
	b.node.MaxSpeed = max
	return b
}

// Build 构建节点
func (b *NodeBuilder) Build() Node {
	return b.node
}

// TestResult 测试结果
type TestResult struct {
	Node    Node
	Error   error
	Latency time.Duration
	Speed   int64
}

// Tester 测试器接口
type Tester interface {
	TestNode(ctx context.Context, out adapter.Outbound) (Node, error)
	TestAll(ctx context.Context) ([]Node, error)
	GetStats() TestStats
	Close() error
}

// TestStats 测试统计
type TestStats struct {
	TotalNodes   int
	SuccessNodes int
	FailedNodes  int
	TotalLatency time.Duration
	AverageSpeed int64
	TestDuration time.Duration
	StartTime    time.Time
	EndTime      time.Time
}

// TestCallback 测试回调接口
type TestCallback interface {
	OnProgress(current, total int)
	OnNodeTested(node Node)
	OnError(err error)
}

// TestOption 测试选项
type TestOption struct {
	Timeout    time.Duration
	Retries    int
	GetCountry bool
	RemoteIP   bool
}
