package ping

// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"sync/atomic"

// 	"github.com/sagernet/sing-box/adapter"
// )

// // ProtocolGroup 管理每个协议的测试状态
// type ProtocolGroup struct {
// 	Protocol     string
// 	SuccessCount atomic.Int32
// 	Nodes        []Node
// 	mu           sync.RWMutex
// }

// // GroupTestManager 管理所有协议组的测试
// type GroupTestManager struct {
// 	groups       map[string]*ProtocolGroup
// 	totalSuccess atomic.Int32
// 	mu           sync.RWMutex
// 	ctx          context.Context
// 	cancel       context.CancelFunc
// }

// // NewGroupTestManager 创建新的分组测试管理器
// func NewGroupTestManager() *GroupTestManager {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	return &GroupTestManager{
// 		groups: make(map[string]*ProtocolGroup),
// 		ctx:    ctx,
// 		cancel: cancel,
// 	}
// }

// // getOrCreateGroup 获取或创建协议组
// func (m *GroupTestManager) getOrCreateGroup(protocol string) *ProtocolGroup {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if group, exists := m.groups[protocol]; exists {
// 		return group
// 	}

// 	group := &ProtocolGroup{
// 		Protocol: protocol,
// 		Nodes:    make([]Node, 0),
// 	}
// 	m.groups[protocol] = group
// 	return group
// }

// // TestOutbounds 测试所有出站连接
// func (m *GroupTestManager) TestOutbounds(outbounds []adapter.Outbound, options *ProfileTestOptions) ([]Node, error) {
// 	var wg sync.WaitGroup
// 	resultChan := make(chan Node, len(outbounds))

// 	// 创建测试器
// 	tester := &ProfileTest{
// 		Options:  options,
// 		Outbound: outbounds,
// 		RemoteIP: false, // 可以根据需要设置
// 	}

// 	// 启动测试协程
// 	for _, out := range outbounds {
// 		wg.Add(1)
// 		go func(out adapter.Outbound) {
// 			defer wg.Done()

// 			// 检查是否需要继续测试
// 			if !m.shouldContinueTesting(out.Type()) {
// 				return
// 			}

// 			// 执行测试
// 			node, err := m.testOutbound(tester, out)
// 			if err != nil {
// 				return
// 			}

// 			// 发送结果
// 			select {
// 			case resultChan <- node:
// 			case <-m.ctx.Done():
// 				return
// 			}
// 		}(out)
// 	}

// 	// 等待所有测试完成或提前终止
// 	go func() {
// 		wg.Wait()
// 		close(resultChan)
// 	}()

// 	// 收集结果
// 	return m.collectResults(resultChan)
// }

// // testOutbound 测试单个出站连接
// func (m *GroupTestManager) testOutbound(tester *ProfileTest, out adapter.Outbound) (Node, error) {
// 	node, err := tester.TestNode(m.ctx, out)
// 	if err != nil {
// 		return node, err
// 	}

// 	// 更新计数
// 	group := m.getOrCreateGroup(out.Type())
// 	if node.IsOk {
// 		group.SuccessCount.Add(1)
// 		m.totalSuccess.Add(1)
// 	}

// 	return node, nil
// }

// // shouldContinueTesting 检查是否应该继续测试
// func (m *GroupTestManager) shouldContinueTesting(protocol string) bool {
// 	// 检查总节点数
// 	if m.totalSuccess.Load() >= MaxTotalNodes {
// 		m.cancel() // 取消所有正在进行的测试
// 		return false
// 	}

// 	// 检查协议节点数
// 	group := m.getOrCreateGroup(protocol)
// 	if group.SuccessCount.Load() >= MaxProtocolNodes {
// 		return false
// 	}

// 	return true
// }

// // collectResults 收集并处理测试结果
// func (m *GroupTestManager) collectResults(resultChan <-chan Node) ([]Node, error) {
// 	var allNodes []Node

// 	for node := range resultChan {
// 		group := m.getOrCreateGroup(node.Protocol)

// 		group.mu.Lock()
// 		group.Nodes = append(group.Nodes, node)
// 		group.mu.Unlock()

// 		allNodes = append(allNodes, node)
// 	}

// 	// 按协议分组统计结果
// 	m.logResults()

// 	return allNodes, nil
// }

// // logResults 输出测试结果统计
// func (m *GroupTestManager) logResults() {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	fmt.Println("\nTest Results by Protocol:")
// 	fmt.Println("-------------------------")

// 	for protocol, group := range m.groups {
// 		fmt.Printf("%s: %d successful nodes\n",
// 			protocol, group.SuccessCount.Load())
// 	}

// 	fmt.Printf("\nTotal successful nodes: %d\n",
// 		m.totalSuccess.Load())
// }

// // GetGroupStats 获取分组统计信息
// func (m *GroupTestManager) GetGroupStats() map[string]int32 {
// 	m.mu.RLock()
// 	defer m.mu.RUnlock()

// 	stats := make(map[string]int32)
// 	for protocol, group := range m.groups {
// 		stats[protocol] = group.SuccessCount.Load()
// 	}
// 	return stats
// }

// // Example usage:
// /*
// func main() {
// 	manager := NewGroupTestManager()

// 	// 准备测试配置
// 	options := &ProfileTestOptions{
// 		Concurrency: 10,
// 		Timeout:    5 * time.Second,
// 	}

// 	// 执行测试
// 	results, err := manager.TestOutbounds(outbounds, options)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// 获取统计信息
// 	stats := manager.GetGroupStats()
// 	for protocol, count := range stats {
// 		fmt.Printf("%s: %d nodes\n", protocol, count)
// 	}
// }
// */
