package job

import (
	"context"
	"sync"
	"time"

	"github.com/nekoimi/scrapio/internal/repo/workflow_repo"
	log "github.com/sirupsen/logrus"
)

type WorkflowScheduler struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWorkflowScheduler() *WorkflowScheduler { return &WorkflowScheduler{} }
func (*WorkflowScheduler) Name() string        { return "WorkflowScheduler" }
func (w *WorkflowScheduler) Start(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			for i := 0; i < 20; i++ {
				processed, err := workflow_repo.DispatchScheduleTick(time.Now())
				if err != nil {
					log.Errorf("工作流调度失败: %v", err)
					break
				}
				if !processed {
					break
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return nil
}
func (w *WorkflowScheduler) Stop(context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	return nil
}
