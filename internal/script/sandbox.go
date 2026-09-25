package script

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
)

const (
	DefaultTimeout        = 2 * time.Second
	DefaultMaxInputBytes  = 512 * 1024
	DefaultMaxOutputBytes = 512 * 1024
)

type Request struct {
	Script         string
	Input          any
	Timeout        time.Duration
	MaxInputBytes  int
	MaxOutputBytes int
}

type Result struct{ Output any }

// Execute runs a pure JavaScript transform. Only the input value is exposed;
// no filesystem, network, process, timer, or Go host object is registered.
func Execute(ctx context.Context, request Request) (Result, error) {
	if strings.TrimSpace(request.Script) == "" {
		return Result{}, errors.New("script is required")
	}
	if request.Timeout <= 0 {
		request.Timeout = DefaultTimeout
	}
	if request.MaxInputBytes <= 0 {
		request.MaxInputBytes = DefaultMaxInputBytes
	}
	if request.MaxOutputBytes <= 0 {
		request.MaxOutputBytes = DefaultMaxOutputBytes
	}
	input, err := json.Marshal(request.Input)
	if err != nil {
		return Result{}, fmt.Errorf("encode script input: %w", err)
	}
	if len(input) > request.MaxInputBytes {
		return Result{}, errors.New("script input exceeds size limit")
	}
	runtime := goja.New()
	var interrupted atomic.Bool
	runtime.Set("input", request.Input)
	// Dynamic code loading is disabled; the supplied script is the only code
	// evaluated by this runtime.
	_ = runtime.Set("eval", nil)
	_ = runtime.Set("Function", nil)
	timer := time.AfterFunc(request.Timeout, func() { interrupted.Store(true); runtime.Interrupt("script timeout") })
	defer timer.Stop()
	if ctx == nil {
		ctx = context.Background()
	}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			runtime.Interrupt("script cancelled")
		case <-stop:
		}
	}()
	value, err := runtime.RunString("(function(input) {\n" + request.Script + "\n})(input)")
	if err != nil {
		if interrupted.Load() {
			return Result{}, context.DeadlineExceeded
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, fmt.Errorf("execute script: %w", err)
	}
	output := value.Export()
	encoded, err := json.Marshal(output)
	if err != nil {
		return Result{}, fmt.Errorf("encode script output: %w", err)
	}
	if len(encoded) > request.MaxOutputBytes {
		return Result{}, errors.New("script output exceeds size limit")
	}
	return Result{Output: output}, nil
}
