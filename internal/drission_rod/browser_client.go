package drission_rod

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	pb "github.com/nekoimi/scrapio/internal/drission_rod/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const BrowserProtocolVersion = "browser-job.v1"

type BrowserAction struct {
	Type     string
	Selector string
	Value    string
	Script   string
	Timeout  time.Duration
}

type BrowserJob struct {
	RequestID string
	URL       string
	Profile   string
	Recipe    string
	Timeout   time.Duration
	Actions   []BrowserAction
	Outputs   []string
	Headers   map[string]string
	ClosePage bool
}

type BrowserResult struct {
	RequestID  string
	HTML       string
	Text       string
	JSON       string
	Screenshot []byte
	Cookies    []*pb.BrowserCookie
	Duration   time.Duration
	DocumentID int64
}

type BrowserError struct {
	Code      string
	RequestID string
	Retryable bool
	Message   string
	Cause     error
}

func (e *BrowserError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("browser job %s: %s", e.Code, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("browser job %s: %v", e.Code, e.Cause)
	}
	return "browser job " + e.Code
}
func (e *BrowserError) Unwrap() error { return e.Cause }

func (d *DrissionRod) Execute(ctx context.Context, job BrowserJob) (BrowserResult, error) {
	if strings.TrimSpace(job.URL) == "" {
		return BrowserResult{}, &BrowserError{Code: "INVALID_JOB", Message: "url is required"}
	}
	if job.RequestID == "" {
		if requestID, ok := ctx.Value("request_id").(string); ok && strings.TrimSpace(requestID) != "" {
			job.RequestID = requestID
		} else {
			job.RequestID = uuid.NewString()
		}
	}
	if job.Timeout <= 0 {
		job.Timeout = 60 * time.Second
	}
	if len(job.Outputs) == 0 {
		job.Outputs = []string{"html"}
	}
	client := d.Client()
	if client == nil {
		return BrowserResult{}, &BrowserError{Code: "BROWSER_UNAVAILABLE", RequestID: job.RequestID, Retryable: true, Message: "browser client is not connected"}
	}

	request := &pb.BrowserJobRequest{ProtocolVersion: BrowserProtocolVersion, RequestId: job.RequestID, Url: job.URL, Profile: job.Profile, Recipe: job.Recipe, TimeoutMs: int32(job.Timeout.Milliseconds()), Outputs: job.Outputs, Headers: job.Headers, ClosePage: job.ClosePage}
	for _, action := range job.Actions {
		request.Actions = append(request.Actions, &pb.BrowserAction{Type: action.Type, Selector: action.Selector, Value: action.Value, Script: action.Script, TimeoutMs: int32(action.Timeout.Milliseconds())})
	}

	var last error
	for attempt := 0; attempt < 3; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, job.Timeout)
		response, err := client.Execute(callCtx, request)
		cancel()
		if err != nil {
			last = err
			if !retryableRPC(err) || ctx.Err() != nil {
				return BrowserResult{}, classifyRPCError(job.RequestID, err)
			}
			if !waitRetry(ctx, time.Duration(attempt+1)*100*time.Millisecond) {
				return BrowserResult{}, classifyRPCError(job.RequestID, ctx.Err())
			}
			continue
		}
		if !response.Success {
			be := &BrowserError{Code: response.ErrorCode, RequestID: response.RequestId, Message: response.Error}
			if be.Code == "" {
				be.Code = "EXECUTION_FAILED"
			}
			be.Retryable = be.Code == "TIMEOUT" || be.Code == "BROWSER_UNAVAILABLE"
			if be.Retryable && attempt < 2 {
				last = be
				if !waitRetry(ctx, time.Duration(attempt+1)*100*time.Millisecond) {
					return BrowserResult{}, classifyRPCError(job.RequestID, ctx.Err())
				}
				continue
			}
			return BrowserResult{}, be
		}
		return BrowserResult{RequestID: response.RequestId, HTML: response.Html, Text: response.Text, JSON: response.Json, Screenshot: response.Screenshot, Cookies: response.Cookies, Duration: time.Duration(response.DurationMs) * time.Millisecond}, nil
	}
	return BrowserResult{}, classifyRPCError(job.RequestID, last)
}

func waitRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (d *DrissionRod) Health(ctx context.Context) error {
	client := d.Client()
	if client == nil {
		return &BrowserError{Code: "BROWSER_UNAVAILABLE", Retryable: true, Message: "browser client is not connected"}
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := client.Health(callCtx, &pb.BrowserHealthRequest{ProtocolVersion: BrowserProtocolVersion, RequestId: uuid.NewString()})
	if err != nil {
		return classifyRPCError("", err)
	}
	if !response.Ready {
		return &BrowserError{Code: "BROWSER_UNAVAILABLE", Message: response.Message, Retryable: true}
	}
	return nil
}

func retryableRPC(err error) bool {
	code := status.Code(err)
	return code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.ResourceExhausted
}
func classifyRPCError(requestID string, err error) error {
	if err == nil {
		return errors.New("browser job failed")
	}
	code := "RPC_ERROR"
	retryable := retryableRPC(err)
	if status.Code(err) == codes.DeadlineExceeded {
		code = "TIMEOUT"
	}
	if status.Code(err) == codes.Canceled {
		code = "CANCELLED"
		retryable = false
	}
	return &BrowserError{Code: code, RequestID: requestID, Retryable: retryable, Message: err.Error(), Cause: err}
}
