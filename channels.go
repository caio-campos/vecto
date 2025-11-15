package vecto

import (
	"context"
)

type channelDispatcher struct {
	logger Logger
}

func newChannelDispatcher(logger Logger) *channelDispatcher {
	return &channelDispatcher{
		logger: logger,
	}
}

func (d *channelDispatcher) dispatch(ctx context.Context, res *Response) {
	responseCopy := res.deepCopy()

	event := RequestCompletedEvent{
		response: responseCopy,
	}

	res.request.mu.RLock()
	channels := make([]chan<- RequestCompletedEvent, len(res.request.events.channels))
	copy(channels, res.request.events.channels)
	res.request.mu.RUnlock()

	if len(channels) == 0 {
		return
	}

	for _, ch := range channels {
		select {
		case <-ctx.Done():
			return
		case ch <- event:
		default:
			if !d.logger.IsNoop() {
				d.logger.Warn(ctx, "channel receiver not ready, skipping event", map[string]interface{}{
					"url": event.response.request.FullUrl(),
				})
			}
		}
	}
}

