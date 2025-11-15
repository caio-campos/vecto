package vecto

import (
	"context"
	"fmt"
	"time"
)

type callbackDispatcher struct {
	logger Logger
	config Config
}

func newCallbackDispatcher(logger Logger, config Config) *callbackDispatcher {
	return &callbackDispatcher{
		logger: logger,
		config: config,
	}
}

func (d *callbackDispatcher) dispatch(ctx context.Context, res *Response) {
	responseCopy := res.deepCopy()

	event := RequestCompletedEvent{
		response: responseCopy,
	}

	requestUrl := responseCopy.request.FullUrl()

	res.request.mu.RLock()
	callbacks := make([]RequestCompletedCallback, len(res.request.events.completed))
	copy(callbacks, res.request.events.completed)
	res.request.mu.RUnlock()

	if len(callbacks) == 0 {
		return
	}

	maxConcurrent := d.config.MaxConcurrentCallbacks
	if maxConcurrent <= 0 {
		maxConcurrent = 100
	}

	callbackTimeout := d.config.CallbackTimeout
	if callbackTimeout <= 0 {
		callbackTimeout = 30 * time.Second
	}

	sem := make(chan struct{}, maxConcurrent)

	for _, cb := range callbacks {
		if ctx.Err() != nil {
			if !d.logger.IsNoop() {
				d.logger.Warn(ctx, "context cancelled, skipping remaining callbacks", map[string]interface{}{
					"url": requestUrl,
				})
			}
			return
		}

		select {
		case <-ctx.Done():
			if !d.logger.IsNoop() {
				d.logger.Warn(ctx, "context cancelled, skipping remaining callbacks", map[string]interface{}{
					"url": requestUrl,
				})
			}
			return
		case sem <- struct{}{}:
		}

		go d.executeCallback(ctx, cb, event, requestUrl, callbackTimeout, sem)
	}
}

func (d *callbackDispatcher) executeCallback(
	ctx context.Context,
	callback RequestCompletedCallback,
	event RequestCompletedEvent,
	url string,
	timeout time.Duration,
	sem chan struct{},
) {
	defer func() {
		<-sem
		if r := recover(); r != nil {
			if !d.logger.IsNoop() {
				d.logger.Error(ctx, "panic in request completed callback", map[string]interface{}{
					"panic": fmt.Sprintf("%v", r),
					"url":   url,
				})
			}
		}
	}()

	callbackCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-callbackCtx.Done():
			return
		default:
			callback(event)
		}
	}()

	select {
	case <-done:
	case <-callbackCtx.Done():
		if !d.logger.IsNoop() {
			d.logger.Error(ctx, "callback timeout exceeded", map[string]interface{}{
				"url":     url,
				"timeout": timeout.String(),
			})
		}
	}
}
