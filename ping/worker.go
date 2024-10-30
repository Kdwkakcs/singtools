package ping

import (
	"log"
	"os"
	"sync"
)

// WorkerPool 工作池
type WorkerPool struct {
	workers chan struct{}
	wg      sync.WaitGroup
	logger  *log.Logger
}

// NewWorkerPool 创建工作池
func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{
		workers: make(chan struct{}, size),
		logger:  log.New(os.Stdout, "[WorkerPool] ", log.LstdFlags),
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(task func()) {
	p.workers <- struct{}{} // 获取工作槽

	go func() {
		defer func() {
			if r := recover(); r != nil {
				if p.logger != nil {
					p.logger.Printf("Recovered from panic in worker: %v", r)
				}
			}
			<-p.workers // 释放工作槽
		}()

		task()
	}()
}

// Wait 等待所有任务完成
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}
