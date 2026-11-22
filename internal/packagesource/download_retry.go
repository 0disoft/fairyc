package packagesource

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	sourceGETAttempts = 3
	sourceRetryDelay  = 200 * time.Millisecond
)

type sourceHTTPStatusError struct{ Code int }

func (err *sourceHTTPStatusError) Error() string {
	return fmt.Sprintf("source response status rejected: HTTP %d", err.Code)
}

// This wrapper admits only bodyless GETs. It cannot replay package publication
// or any external write. Returned bodies belong to the caller; discarded retry
// responses are closed here before backoff, without reading untrusted payloads.
func requestSourceGET(client *http.Client, req *http.Request, wait func(context.Context, time.Duration) error) (*http.Response, error) {
	if req.Method != http.MethodGet || req.Body != nil {
		return nil, fmt.Errorf("source retry requires a bodyless GET")
	}
	ctx := req.Context()
	var lastErr error
	for attempt := 0; ; attempt++ {
		if ctx.Err() != nil {
			return nil, errors.Join(lastErr, ctx.Err(), context.Cause(ctx))
		}
		response, err := client.Do(req)
		if ctx.Err() != nil {
			if response != nil {
				err = errors.Join(err, response.Body.Close())
			}
			return nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
		}
		retry := transientSourceError(err)
		if err == nil {
			if !transientSourceStatus(response.StatusCode) || response.Header.Get("Retry-After") != "" {
				return response, nil
			}
			err = &sourceHTTPStatusError{response.StatusCode}
			if closeErr := response.Body.Close(); closeErr != nil {
				return nil, errors.Join(err, fmt.Errorf("close source retry response: %w", closeErr))
			}
			retry = true
		}
		if !retry || attempt == sourceGETAttempts-1 {
			return nil, err
		}
		lastErr = err
		if err := wait(ctx, sourceRetryDelay<<attempt); err != nil {
			return nil, errors.Join(lastErr, err, ctx.Err(), context.Cause(ctx))
		}
	}
}

func transientSourceStatus(code int) bool {
	return code == http.StatusBadGateway || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout
}

func transientSourceError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	// Client.Do wraps transport errors in a URL error. A transport timeout
	// may match DeadlineExceeded even while the shared request context is live.
	var requestErr *url.Error
	if errors.As(err, &requestErr) {
		err = requestErr.Err
	}
	var network net.Error
	return errors.As(err, &network) && network != context.DeadlineExceeded && (network.Timeout() || network.Temporary())
}

func waitSourceRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
