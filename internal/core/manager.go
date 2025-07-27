package core

import (
	"context"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// LifecycleManager 负责统一管理模块的启动与关闭
type LifecycleManager struct {
	// context
	ctx context.Context
	// context取消函数
	cancel context.CancelFunc
	// signal
	sigs chan os.Signal
	// 管理的应用列表
	lifecycles []Lifecycle
}

func NewLifecycleManager(parent context.Context) *LifecycleManager {
	ctx, cancel := context.WithCancel(parent)
	m := &LifecycleManager{
		ctx:        ctx,
		cancel:     cancel,
		sigs:       make(chan os.Signal, 1),
		lifecycles: make([]Lifecycle, 0),
	}
	signal.Notify(m.sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	return m
}

func (m *LifecycleManager) Register(lifecycle Lifecycle) {
	m.lifecycles = append(m.lifecycles, lifecycle)
}

func (m *LifecycleManager) StartAndWaiting() {
	for _, lifecycle := range m.lifecycles {
		go func(life Lifecycle) {
			log.Infof("[Lifecycle] Starting: %s ...", life.Name())
			if err := life.Start(m.ctx); err != nil {
				log.Errorf("[Lifecycle] failed to start %s: %s", life.Name(), err)
			}
		}(lifecycle)
	}
	m.waitForSignal()
	m.shutdown(30 * time.Second)
}

func (m *LifecycleManager) waitForSignal() {
	sig := <-m.sigs
	log.Infof("收到退出信号: %v，正在关闭...", sig)
	m.cancel()
}

func (m *LifecycleManager) shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	wait := sync.WaitGroup{}
	for _, lifecycle := range m.lifecycles {
		wait.Add(1)
		go func(life Lifecycle) {
			defer wait.Done()
			log.Infof("[Lifecycle] Stopping: %s ...", life.Name())
			if err := life.Stop(ctx); err != nil {
				log.Errorf("[Lifecycle] Stop error in %s: %v\n", life.Name(), err)
			}
		}(lifecycle)
	}

	wait.Wait()
	log.Infoln("Exit.")
}
