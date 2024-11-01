package ping

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Dkwkoaca/singtools/log"
	"github.com/sagernet/sing-box/adapter"
)

// Service 测试服务
type Service struct {
	config        *Config
	testManager   *TestManager
	geoManager    *GeoIPManager
	resultManager *ResultManager
	logger        *log.Logger
}

// NewService 创建测试服务
func NewService(config *Config) (*Service, error) {
	// 初始化配置
	config.InitWithDefaults()

	// 初始化日志
	logger := log.NewLogger("info")

	// 创建测试管理器
	testManager, err := NewTestManager(config)
	if err != nil {
		return nil, err
	}

	// 创建地理位置管理器
	geoManager, err := NewGeoIPManager(config)
	if err != nil {
		return nil, err
	}

	fmt.Printf("config: %+v\n", config)
	return &Service{
		config:        config,
		testManager:   testManager,
		geoManager:    geoManager,
		resultManager: NewResultManager(config),
		logger:        logger,
	}, nil
}

// RunTests 执行测试
func (s *Service) RunTests(ctx context.Context, outbounds []adapter.Outbound) ([]Node, error) {
	start := time.Now()
	s.logger.Info(fmt.Sprintf("Starting tests for %d nodes", len(outbounds)))

	// 执行测试
	nodes, err := s.testManager.TestAll(ctx, outbounds)
	if err != nil {
		fmt.Println("Error running tests:", err)
		return nil, err
	}
	// fmt.Println(nodes)
	strs, _ := json.Marshal(nodes)
	s.logger.Debug(string(strs))
	// 处理结果
	for _, node := range nodes {
		s.resultManager.AddNode(node)
	}

	// 去重
	if s.config.RemoveDup {
		nodes = s.resultManager.RemoveDuplicates()
	}

	// 排序
	nodes = s.resultManager.SortNodes(s.config.SortMethod)

	duration := time.Since(start)
	stats := s.testManager.metrics.GetStats()

	// 安全地输出统计信息
	s.logger.Info(fmt.Sprintf("Tests completed in %v", duration))
	s.logger.Info(fmt.Sprintf("Total nodes: %d", stats.TotalNodes))
	s.logger.Info(fmt.Sprintf("Successful nodes: %d", stats.SuccessNodes))
	s.logger.Info(fmt.Sprintf("Failed nodes: %d", stats.FailedNodes))
	s.logger.Info(fmt.Sprintf("Success rate: %.2f%%", stats.SuccessRate))

	return nodes, nil
}

// Close 关闭服务
func (s *Service) Close() error {
	if err := s.geoManager.Close(); err != nil {
		s.logger.Info(fmt.Sprintf("Error closing GeoIP manager: %v", err))
	}
	return nil
}

func (s *Service) Config() Config {
	return *s.config
}
