package packagesource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

// Fetcher optionally accepts a caller-owned client for TLS trust configuration, not ambient auth.
// The CLI always uses the default verified TLS client; redirects/time limits remain enforced here.
type Fetcher struct{ Client *http.Client }

const sourceDownloadTimeout = 30 * time.Second

// HTTP I/O is already supplied by Fetcher.Client. Wait is a per-download port
// for cancellable backoff; nil retains the host timer. The existing context
// argument uses Background when nil. Overrides must honor that context.
type sourceDownloadOptions struct {
	Wait func(context.Context, time.Duration) error
}

func newSourceHTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, ForceAttemptHTTP2: true, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second, DisableCompression: true}}
}

func (f Fetcher) download(ctx context.Context, pin Pin) ([]byte, error) {
	return f.downloadWithin(ctx, pin, MaxArchiveBytes)
}

func (f Fetcher) downloadWithin(ctx context.Context, pin Pin, limit int, options ...sourceDownloadOptions) ([]byte, error) {
	if len(options) > 1 {
		return nil, fmt.Errorf("at most one source download options value is supported")
	}
	wait := waitSourceRetry
	if len(options) == 1 && options[0].Wait != nil {
		wait = options[0].Wait
	}
	if limit <= 0 || limit > MaxArchiveBytes {
		return nil, fmt.Errorf("source compressed byte budget exhausted")
	}
	if err := validateURL(pin.URL); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	// One budget covers every GET attempt, backoff and the final body read.
	ctx, cancel := context.WithTimeout(ctx, sourceDownloadTimeout)
	defer cancel()
	shared := f.Client
	if shared == nil {
		shared = newSourceHTTPClient()
		defer shared.CloseIdleConnections()
	}
	client := *shared
	// A caller client supplies transport/TLS configuration, not user session
	// state. Isolate both Cookie attachment and Set-Cookie persistence without
	// mutating the caller's client or its shared connection pool.
	client.Jar = nil
	client.Timeout = sourceDownloadTimeout
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pin.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/zip")
	response, err := requestSourceGET(&client, req, wait)
	if err != nil {
		return nil, fmt.Errorf("source HTTPS request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, &sourceHTTPStatusError{response.StatusCode}
	}
	if response.ContentLength > int64(limit) || response.Uncompressed || (response.Header.Get("Content-Encoding") != "" && response.Header.Get("Content-Encoding") != "identity") {
		return nil, fmt.Errorf("source response status, encoding or size rejected")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil || ctx.Err() != nil {
		return nil, fmt.Errorf("read source archive: %w", errors.Join(err, ctx.Err(), context.Cause(ctx)))
	}
	if len(data) > limit {
		return nil, fmt.Errorf("source download byte limit exceeded")
	}
	return data, nil
}

func (f Fetcher) prepare(ctx context.Context, root *os.Root, pin Pin, language string, offline bool) (bundle, []byte, bool, error) {
	return f.prepareWithin(ctx, root, pin, language, offline, MaxArchiveBytes, MaxTotalBytes, MaxFiles)
}

func (f Fetcher) prepareWithin(ctx context.Context, root *os.Root, pin Pin, language string, offline bool, compressed, expanded, files int) (bundle, []byte, bool, error) {
	limit := min(compressed, MaxArchiveBytes)
	if limit <= 0 || expanded <= 0 || files <= 0 {
		return bundle{}, nil, false, fmt.Errorf("source fetch budget exhausted")
	}
	data, err := readRegular(root, path.Join(CachePath, pin.SHA256+".zip"), int64(limit))
	hit := err == nil
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return bundle{}, nil, false, err
		}
		if offline {
			return bundle{}, nil, false, fmt.Errorf("offline source cache miss: %s", pin.Path)
		}
		data, err = f.downloadWithin(ctx, pin, limit)
		if err != nil {
			return bundle{}, nil, false, err
		}
	}
	if hash(data) != pin.SHA256 {
		return bundle{}, nil, false, fmt.Errorf("source archive SHA-256 mismatch: %s", pin.Path)
	}
	b, err := unpackWithin(ctx, data, pin, language, expanded, files)
	return b, data, hit, err
}
