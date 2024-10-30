package ping

import (
	"fmt"
	"sort"
	"sync"
)

// ResultManager 结果管理器
type ResultManager struct {
	nodes  []Node
	mu     sync.RWMutex
	config *Config
}

// NewResultManager 创建结果管理器
func NewResultManager(config *Config) *ResultManager {
	return &ResultManager{
		nodes:  make([]Node, 0),
		config: config,
	}
}

// AddNode 添加节点
func (r *ResultManager) AddNode(node Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes = append(r.nodes, node)
}

// GetNodes 获取所有节点
func (r *ResultManager) GetNodes() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Node, len(r.nodes))
	copy(result, r.nodes)
	return result
}

// SortNodes 节点排序
func (r *ResultManager) SortNodes(method string) []Node {
	nodes := r.GetNodes()

	switch method {
	case "ping":
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Ping < nodes[j].Ping
		})
	case "speed":
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].AvgSpeed > nodes[j].AvgSpeed
		})
	case "country":
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].Country == nodes[j].Country {
				return nodes[i].Ping < nodes[j].Ping
			}
			return nodes[i].Country < nodes[j].Country
		})
	}

	return nodes
}

// FilterNodes 节点过滤
func (r *ResultManager) FilterNodes(filter func(Node) bool) []Node {
	nodes := r.GetNodes()
	result := make([]Node, 0, len(nodes))

	for _, node := range nodes {
		if filter(node) {
			result = append(result, node)
		}
	}

	return result
}

// RemoveDuplicates 删除重复节点
func (r *ResultManager) RemoveDuplicates() []Node {
	nodes := r.GetNodes()
	seen := make(map[string]bool)
	result := make([]Node, 0, len(nodes))

	for _, node := range nodes {
		key := node.generateKey()
		if !seen[key] {
			seen[key] = true
			result = append(result, node)
		}
	}

	return result
}

// generateKey 生成节点唯一键
func (n Node) generateKey() string {
	return fmt.Sprintf("%s:%s:%s", n.Protocol, n.Ip, n.Country)
}
