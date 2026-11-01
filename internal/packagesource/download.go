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

func (f Fetcher) download(ctx context.Context, pin Pin) ([]byte, error) {
	return f.downloadWithin(ctx, pin, MaxArchiveBytes)
}

func (f Fetcher) downloadWithin(ctx context.Context, pin Pin, limit int) ([]byte, error) {
	if limit <= 0 || limit > MaxArchiveBytes {
		return nil, fmt.Errorf("source compressed byte budget exhausted")
	}
	if err := validateURL(pin.URL); err != nil {
		return nil, err
	}
	client := http.Client{Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, ForceAttemptHTTP2: true, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second, DisableCompression: true}}
	if f.Client != nil {
		client = *f.Client
	}
	client.Timeout = 30 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	if f.Client == nil {
		defer client.CloseIdleConnections()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pin.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/zip")
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("source HTTPS request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > int64(limit) || response.Uncompressed || (response.Header.Get("Content-Encoding") != "" && response.Header.Get("Content-Encoding") != "identity") {
		return nil, fmt.Errorf("source response status, encoding or size rejected")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil {
		return nil, err
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
