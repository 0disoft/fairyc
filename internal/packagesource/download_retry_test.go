package packagesource

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSourceDownloadRecoversTransientGET(t *testing.T) {
	data := archiveBytes(t)
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("source retrieval must only issue GET")
		}
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	pin := testPin()
	pin.URL, pin.SHA256 = server.URL+"/math.zip", hash(data)
	dir := fixtureWorkspace(t, pin)
	result, err := (Fetcher{Client: server.Client()}).Fetch(context.Background(), dir, false)
	if err != nil || result.Installed != 1 || result.Downloaded != 1 || calls.Load() != 3 {
		t.Fatalf("transient GET recovery: %+v, calls=%d, err=%v", result, calls.Load(), err)
	}
	if _, err := os.Stat(dir + "/" + fetchLock); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fetch lock survived recovery: %v", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	cached, err := readRegular(root, CachePath+"/"+pin.SHA256+".zip", MaxArchiveBytes)
	if err != nil || !bytes.Equal(cached, data) {
		t.Fatalf("recovered cache bytes changed: %v", err)
	}
}

func TestSourceDownloadReportsHTTPStatus(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	pin := testPin()
	pin.URL = server.URL + "/math.zip"
	_, err := (Fetcher{Client: server.Client()}).download(context.Background(), pin)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("HTTP failure lost its status: %v", err)
	}
}

type sourceRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn sourceRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

type sourceRetryBody struct {
	io.Reader
	close func() error
}

func (body sourceRetryBody) Close() error { return body.close() }

func TestSourceGETStatusRetryPolicy(t *testing.T) {
	for _, status := range []int{200, 302, 400, 401, 403, 404, 429, 500, 502, 503, 504} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls, closed := 0, 0
			var delays []time.Duration
			client := &http.Client{Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if calls != closed {
					t.Error("previous response remained open on retry")
				}
				calls++
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: sourceRetryBody{
					Reader: strings.NewReader(""), close: func() error { closed++; return nil },
				}}, nil
			})}
			req, _ := http.NewRequest(http.MethodGet, "https://packages.example/math.zip", nil)
			response, err := requestSourceGET(client, req, func(ctx context.Context, delay time.Duration) error {
				if calls != closed {
					t.Error("retry response remained open during backoff")
				}
				delays = append(delays, delay)
				return nil
			})
			if response != nil {
				_ = response.Body.Close()
			}
			wantCalls := 1
			if status == 502 || status == 503 || status == 504 {
				wantCalls = 3
				var failure *sourceHTTPStatusError
				if !errors.As(err, &failure) || failure.Code != status || !slices.Equal(delays, []time.Duration{200 * time.Millisecond, 400 * time.Millisecond}) {
					t.Fatalf("retry diagnostics or backoff: %v, %v", err, delays)
				}
			} else if err != nil || len(delays) != 0 {
				t.Fatalf("retried a terminal response: %v, %v", err, delays)
			}
			if calls != wantCalls || closed != calls {
				t.Fatalf("calls=%d, closed=%d, want %d", calls, closed, wantCalls)
			}
		})
	}
}

func TestSourceGETDoesNotOverrideRetryAfterOrReplayWrites(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 503, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	wait := func(context.Context, time.Duration) error { t.Error("unexpected retry"); return nil }
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req, _ := http.NewRequest(method, "https://packages.example/math.zip", nil)
		if _, err := requestSourceGET(client, req, wait); err == nil || calls != 0 {
			t.Fatalf("GET-only helper admitted %s: calls=%d, err=%v", method, calls, err)
		}
	}
	req, _ := http.NewRequest(http.MethodGet, "https://packages.example/math.zip", nil)
	response, err := requestSourceGET(client, req, wait)
	if err != nil || response == nil || calls != 1 {
		t.Fatalf("server Retry-After ignored: calls=%d, err=%v", calls, err)
	}
	_ = response.Body.Close()
}

func TestSourceGETCancellationDuringBackoffPreservesCause(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			if deadline {
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, 20*time.Millisecond)
				defer stop()
			}
			calls, closed := 0, false
			client := &http.Client{Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 503, Header: make(http.Header), Body: sourceRetryBody{
					Reader: strings.NewReader(""), close: func() error { closed = true; return nil },
				}}, nil
			})}
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://packages.example/math.zip", nil)
			cause := errors.New("operator stopped download")
			_, err := requestSourceGET(client, req, func(ctx context.Context, delay time.Duration) error {
				if !closed {
					t.Error("backoff retained response body")
				}
				if !deadline {
					cancel(cause)
				}
				return waitSourceRetry(ctx, delay)
			})
			want := error(context.Canceled)
			if deadline {
				want = context.DeadlineExceeded
			} else if !errors.Is(err, cause) {
				t.Fatalf("lost cancellation cause: %v", err)
			}
			var status *sourceHTTPStatusError
			if calls != 1 || !errors.Is(err, want) || !errors.As(err, &status) || status.Code != 503 {
				t.Fatalf("cancelled GET retried or lost original failure: calls=%d, err=%v", calls, err)
			}
		})
	}
}

func TestSourceGETTransportFailuresPreserveCause(t *testing.T) {
	for _, test := range []struct {
		name  string
		err   error
		calls int
	}{
		{"temporary", &net.DNSError{Err: "temporary DNS failure", IsTemporary: true}, 3},
		{"network-timeout", &net.DNSError{Err: "temporary network timeout", IsTimeout: true}, 3},
		{"permanent", errors.New("transport configuration failed"), 1},
		{"TLS", &tls.CertificateVerificationError{Err: errors.New("untrusted certificate")}, 1},
		{"cancelled", context.Canceled, 1},
		{"deadline", context.DeadlineExceeded, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: sourceRoundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, test.err
			})}
			req, _ := http.NewRequest(http.MethodGet, "https://packages.example/math.zip", nil)
			_, err := requestSourceGET(client, req, func(context.Context, time.Duration) error { return nil })
			if !errors.Is(err, test.err) || calls != test.calls {
				t.Fatalf("transport cause or attempt count changed: calls=%d, err=%v", calls, err)
			}
		})
	}
}

type sourceReadFunc func([]byte) (int, error)

func (fn sourceReadFunc) Read(data []byte) (int, error) { return fn(data) }

func TestSourceDownloadBoundsBodyAndPreservesCancellation(t *testing.T) {
	for _, mode := range []string{"partial", "cancelled-partial", "cancelled-complete"} {
		t.Run(mode, func(t *testing.T) {
			cancelled := mode != "partial"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cancelled while reading archive")
			calls, closed := 0, false
			client := &http.Client{Transport: sourceRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				deadline, bounded := req.Context().Deadline()
				if !bounded || time.Until(deadline) > 30*time.Second {
					t.Error("archive body has no shared 30s deadline")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: sourceRetryBody{
					Reader: sourceReadFunc(func([]byte) (int, error) {
						if cancelled {
							cancel(cause)
						}
						if mode == "cancelled-complete" {
							return 0, io.EOF
						}
						return 0, io.ErrUnexpectedEOF
					}), close: func() error { closed = true; return nil },
				}}, nil
			})}
			_, err := (Fetcher{Client: client}).download(ctx, testPin())
			if mode != "cancelled-complete" && !errors.Is(err, io.ErrUnexpectedEOF) || calls != 1 || !closed {
				t.Fatalf("partial download retried, lost cause, or leaked body: calls=%d, closed=%v, err=%v", calls, closed, err)
			}
			if cancelled && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatalf("body cancellation lost its cause: %v", err)
			}
		})
	}
}

func TestSourceGETCancelledBeforeRequestAndResponseCloseFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	closeFailure := errors.New("response close failed")
	client := &http.Client{Transport: sourceRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 503, Header: make(http.Header), Body: sourceRetryBody{
			Reader: strings.NewReader(""), close: func() error { return closeFailure },
		}}, nil
	})}
	wait := func(context.Context, time.Duration) error { t.Error("unexpected retry"); return nil }
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://packages.example/math.zip", nil)
	if _, err := requestSourceGET(client, req, wait); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("cancelled GET performed I/O: calls=%d, err=%v", calls, err)
	}
	req, _ = http.NewRequest(http.MethodGet, "https://packages.example/math.zip", nil)
	if _, err := requestSourceGET(client, req, wait); !errors.Is(err, closeFailure) || calls != 1 {
		t.Fatalf("close failure retried or swallowed: calls=%d, err=%v", calls, err)
	}
}

func TestSourceDownloadCancelsBlockedHTTPSRequest(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(stopped)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	pin := testPin()
	pin.URL = server.URL + "/math.zip"
	done := make(chan error, 1)
	go func() {
		_, err := (Fetcher{Client: server.Client()}).download(ctx, pin)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTPS request did not start")
	}
	cause := errors.New("operator cancelled stalled origin")
	cancel(cause)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatalf("stalled request lost cancellation cause: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stalled request ignored cancellation")
	}
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled request retained the origin connection")
	}
}
