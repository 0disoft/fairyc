package packagesource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

type sourceCookieJar struct{ reads, writes int }

func (jar *sourceCookieJar) Cookies(*url.URL) []*http.Cookie {
	jar.reads++
	return []*http.Cookie{{Name: "local-preference", Value: "dark"}}
}

func (jar *sourceCookieJar) SetCookies(*url.URL, []*http.Cookie) { jar.writes++ }

func TestSourceDownloadCookieIsolationOnRetries(t *testing.T) {
	for _, successAt := range []int{2, 3, 4} {
		t.Run(fmt.Sprint(successAt), func(t *testing.T) {
			jar := &sourceCookieJar{}
			requests, closed := 0, 0
			client := &http.Client{Jar: jar, Timeout: time.Minute, Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if requests != closed || req.Header.Get("Cookie") != "" {
					t.Error("previous body left open or caller cookie transmitted")
				}
				requests++
				status := http.StatusServiceUnavailable
				if requests == successAt {
					status = http.StatusOK
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Set-Cookie": {"remote-preference=light; Path=/; Secure"}},
					Body: sourceRetryBody{Reader: strings.NewReader("archive"), close: func() error { closed++; return nil }}}, nil
			})}
			var delays []time.Duration
			data, err := (Fetcher{Client: client}).downloadWithin(nil, testPin(), 8, sourceDownloadOptions{Wait: func(ctx context.Context, delay time.Duration) error {
				if closed != requests {
					t.Error("retry body open during backoff")
				}
				delays = append(delays, delay)
				return ctx.Err()
			}})
			if requests != min(successAt, 3) || closed != requests || jar.reads != 0 || jar.writes != 0 || client.Jar != jar || client.Timeout != time.Minute {
				t.Fatalf("session isolation failed: requests=%d closed=%d jar=%+v", requests, closed, jar)
			}
			wantDelays := []time.Duration{200 * time.Millisecond, 400 * time.Millisecond}
			if !slices.Equal(delays, wantDelays[:min(successAt-1, 2)]) {
				t.Fatalf("retry delays=%v", delays)
			}
			if successAt <= 3 {
				if err != nil || string(data) != "archive" {
					t.Fatalf("retry failed: %q %v", data, err)
				}
			} else {
				var status *sourceHTTPStatusError
				if data != nil || !errors.As(err, &status) || status.Code != http.StatusServiceUnavailable {
					t.Fatalf("retry budget bypassed: %q %v", data, err)
				}
			}
		})
	}
}

func TestSourceDownloadCookieIsolationByteBoundaries(t *testing.T) {
	for _, size := range []int{7, 8, 9} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			jar := &sourceCookieJar{}
			closed := 0
			client := &http.Client{Jar: jar, Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("Cookie") != "" {
					t.Error("caller cookie transmitted")
				}
				return &http.Response{StatusCode: 200, ContentLength: -1, Header: http.Header{"Set-Cookie": {"remote-preference=light"}},
					Body: sourceRetryBody{Reader: strings.NewReader(strings.Repeat("x", size)), close: func() error { closed++; return nil }}}, nil
			})}
			data, err := (Fetcher{Client: client}).downloadWithin(nil, testPin(), 8)
			if size <= 8 && (err != nil || len(data) != size) || size > 8 && (err == nil || data != nil) {
				t.Fatalf("byte boundary %d: bytes=%d err=%v", size, len(data), err)
			}
			if closed != 1 || jar.reads != 0 || jar.writes != 0 {
				t.Fatalf("body or cookie ownership: closed=%d jar=%+v", closed, jar)
			}
		})
	}
}

func TestSourceDownloadCookieIsolationFailures(t *testing.T) {
	for _, mode := range []string{"transport", "body", "wait", "cancel during wait", "pre-cancelled"} {
		t.Run(mode, func(t *testing.T) {
			jar := &sourceCookieJar{}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			failure := errors.New("download stopped")
			if mode == "pre-cancelled" {
				cancel(failure)
			}
			requests, closed, waits := 0, 0, 0
			client := &http.Client{Jar: jar, Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Header.Get("Cookie") != "" {
					t.Error("caller cookie transmitted")
				}
				if mode == "transport" {
					return nil, failure
				}
				status := http.StatusServiceUnavailable
				var body io.Reader = strings.NewReader("")
				if mode == "body" {
					status = http.StatusOK
					body = sourceReadFunc(func([]byte) (int, error) { return 0, failure })
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Set-Cookie": {"remote-preference=light"}},
					Body: sourceRetryBody{Reader: body, close: func() error { closed++; return nil }}}, nil
			})}
			data, err := (Fetcher{Client: client}).downloadWithin(ctx, testPin(), 8, sourceDownloadOptions{Wait: func(context.Context, time.Duration) error {
				waits++
				if mode == "cancel during wait" {
					cancel(failure)
					return ctx.Err()
				}
				return failure
			}})
			if data != nil || !errors.Is(err, failure) || jar.reads != 0 || jar.writes != 0 || client.Jar != jar {
				t.Fatalf("lost error or touched cookie state: jar=%+v err=%v", jar, err)
			}
			wantRequests, wantClosed, wantWaits := 1, 1, 0
			switch mode {
			case "pre-cancelled":
				wantRequests, wantClosed = 0, 0
			case "transport":
				wantClosed = 0
			case "wait", "cancel during wait":
				wantWaits = 1
			}
			if requests != wantRequests || closed != wantClosed || waits != wantWaits {
				t.Fatalf("unexpected I/O: requests=%d closed=%d waits=%d", requests, closed, waits)
			}
			if ctx.Err() != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost cancellation: %v", err)
			}
		})
	}
}

func TestSourceDownloadPolicyRejectsBeforeIO(t *testing.T) {
	for _, mode := range []string{"zero budget", "excess budget", "extra options"} {
		t.Run(mode, func(t *testing.T) {
			jar := &sourceCookieJar{}
			requests := 0
			client := &http.Client{Jar: jar, Transport: sourceRoundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return nil, errors.New("unexpected request")
			})}
			limit := 8
			var options []sourceDownloadOptions
			switch mode {
			case "zero budget":
				limit = 0
			case "excess budget":
				limit = MaxArchiveBytes + 1
			case "extra options":
				options = []sourceDownloadOptions{{}, {}}
			}
			data, err := (Fetcher{Client: client}).downloadWithin(nil, testPin(), limit, options...)
			if data != nil || err == nil || requests != 0 || jar.reads != 0 || jar.writes != 0 {
				t.Fatalf("invalid policy performed I/O: requests=%d jar=%+v err=%v", requests, jar, err)
			}
		})
	}
}

func TestSourceDownloadDoesNotUseCallerCookieJar(t *testing.T) {
	jar := &sourceCookieJar{}
	requests, cookieRequests := 0, 0
	client := &http.Client{Jar: jar, Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Header.Get("Cookie") != "" {
			cookieRequests++
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Set-Cookie": {"remote-preference=light; Path=/; Secure"}}, Body: io.NopCloser(strings.NewReader("archive"))}, nil
	})}
	data, err := (Fetcher{Client: client}).download(context.Background(), testPin())
	if err != nil || string(data) != "archive" || requests != 1 {
		t.Fatalf("download failed: requests=%d err=%v", requests, err)
	}
	if cookieRequests != 0 || jar.reads != 0 || jar.writes != 0 {
		t.Fatalf("download used caller cookie state: sent=%d reads=%d writes=%d", cookieRequests, jar.reads, jar.writes)
	}
	if client.Jar != jar {
		t.Fatal("download modified the caller client")
	}
}
