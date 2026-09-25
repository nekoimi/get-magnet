package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nekoimi/get-magnet/internal/bean"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/repo/plugin_repo"
	log "github.com/sirupsen/logrus"
)

const (
	LeaseDuration        = 5 * time.Minute
	PollInterval         = 500 * time.Millisecond
	ExternalPollInterval = 10 * time.Second
)

type Worker struct {
	registry *Registry
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	count    int
}

func NewWorker(registry *Registry) *Worker { return &Worker{registry: registry} }
func (w *Worker) Name() string             { return "PluginWorker" }

func (w *Worker) Start(parent context.Context) error {
	if w.registry == nil {
		return errors.New("plugin registry is required")
	}
	cfg := bean.PtrFromContext[config.Config](parent)
	w.count = 1
	if cfg != nil && cfg.Crawler != nil && cfg.Crawler.WorkerNum > 0 {
		w.count = cfg.Crawler.WorkerNum
	}
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	for i := 0; i < w.count; i++ {
		w.wg.Add(1)
		go w.loop(ctx, i)
	}
	return nil
}

func (w *Worker) Stop(_ context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	return nil
}

func (w *Worker) loop(ctx context.Context, index int) {
	defer w.wg.Done()
	workerID := fmt.Sprintf("plugin-%d-%s", index, uuid.NewString()[:8])
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		claim, found, err := plugin_repo.ClaimNext(workerID, LeaseDuration)
		if err != nil && !pluginRepoUnavailable(err) {
			log.Warnf("领取插件任务失败: %s", err)
		}
		if err != nil || !found {
			if !wait(ctx, PollInterval) {
				return
			}
			continue
		}
		handler, ok := w.registry.Get(claim.Task.PluginCode)
		if !ok {
			_ = plugin_repo.Fail(claim.Task.Id, fmt.Errorf("plugin %q is not registered", claim.Task.PluginCode), false)
			continue
		}
		input := map[string]any{}
		if err := json.Unmarshal([]byte(claim.Task.Input), &input); err != nil {
			_ = plugin_repo.Fail(claim.Task.Id, err, false)
			continue
		}
		task := Task{ResourceID: claim.Task.ResourceId, EventType: claim.Task.EventType, Input: input}
		async, isAsync := handler.(AsyncHandler)
		if isAsync && claim.Task.ExternalID != "" {
			output, done, err := async.Poll(ctx, task, claim.Task.ExternalID)
			if err != nil {
				_ = plugin_repo.Fail(claim.Task.Id, err, !isPermanent(err))
				continue
			}
			if !done {
				if err := plugin_repo.SchedulePoll(claim.Task.Id, output, claim.Task.ExternalID, ExternalPollInterval); err != nil {
					log.Errorf("重新调度插件轮询失败: %s", err)
				}
				continue
			}
			if completion, ok := handler.(CompletionHandler); ok {
				if err := completion.OnComplete(ctx, task, output); err != nil {
					_ = plugin_repo.Fail(claim.Task.Id, err, true)
					continue
				}
			}
			if err := plugin_repo.Complete(claim.Task.Id, output, claim.Task.ExternalID); err != nil {
				log.Errorf("完成插件任务失败: %s", err)
			}
			continue
		}

		output, externalID, err := handler.Handle(ctx, task)
		if err != nil {
			_ = plugin_repo.Fail(claim.Task.Id, err, !isPermanent(err))
			continue
		}
		if isAsync && externalID != "" {
			if err := plugin_repo.SchedulePoll(claim.Task.Id, output, externalID, ExternalPollInterval); err != nil {
				log.Errorf("调度插件轮询失败: %s", err)
			}
			continue
		}
		if completion, ok := handler.(CompletionHandler); ok {
			if err := completion.OnComplete(ctx, task, output); err != nil {
				_ = plugin_repo.Fail(claim.Task.Id, err, true)
				continue
			}
		}
		if err := plugin_repo.Complete(claim.Task.Id, output, externalID); err != nil {
			log.Errorf("完成插件任务失败: %s", err)
		}
	}
}

func isPermanent(err error) bool {
	var permanent *PermanentError
	return errors.As(err, &permanent)
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func pluginRepoUnavailable(err error) bool {
	return err != nil && err.Error() == "database is not initialized"
}
