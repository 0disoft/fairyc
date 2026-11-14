package packagesource

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const fetchLock = ".fairyc-fetch.lock"

type Result struct {
	Packages, Downloaded, CacheHits, Installed, Reused int
}

// Fetch prepares exact local members. It never runs package code or updates either manifest/lock.
func (f Fetcher) Fetch(ctx context.Context, workspace string, offline bool) (result Result, err error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	info, err := os.Lstat(workspace)
	if err != nil {
		return result, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return result, fmt.Errorf("source workspace must be a non-symlink directory")
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return result, err
	}
	defer root.Close()
	data, err := readRegular(root, FileName, MaxPlanBytes)
	if err != nil {
		return result, err
	}
	plan, err := Parse(data)
	if err != nil {
		return result, err
	}
	data, err = readRegular(root, projectmanifest.FileName, projectmanifest.MaxBytes)
	if err != nil {
		return result, err
	}
	manifest, err := projectmanifest.Parse(data)
	if err != nil {
		return result, err
	}
	if err := Bind(plan, manifest); err != nil {
		return result, err
	}
	if err := root.Mkdir(fetchLock, 0700); err != nil {
		return result, fmt.Errorf("cannot acquire exclusive package fetch lock: %w", err)
	}
	defer func() { err = errors.Join(err, root.Remove(fetchLock)) }()
	// The value receiver owns this operation's default connection pool. Reuse it
	// across package downloads, then close it once; never close caller clients.
	if !offline && f.Client == nil {
		f.Client = newSourceHTTPClient()
		defer f.Client.CloseIdleConnections()
	}
	var compressed, expanded, files int
	for _, pin := range plan.Packages {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		b, data, hit, err := f.prepareWithin(ctx, root, pin, manifest.Package.Language, offline, MaxTotalBytes-compressed, MaxTotalBytes-expanded, MaxFiles-files)
		if err != nil {
			return result, fmt.Errorf("fetch %s: %w", pin.Path, err)
		}
		compressed += len(data)
		expanded += b.bytes
		files += len(b.files)
		if compressed > MaxTotalBytes || expanded > MaxTotalBytes || files > MaxFiles {
			return result, fmt.Errorf("combined source fetch byte/file limit exceeded")
		}
		if !hit {
			if err := storeArchive(root, pin, data); err != nil {
				return result, err
			}
			result.Downloaded++
		} else {
			result.CacheHits++
		}
		installed, err := publish(ctx, root, pin, b)
		if err != nil {
			return result, fmt.Errorf("publish %s: %w", pin.Path, err)
		}
		if installed {
			result.Installed++
		} else {
			result.Reused++
		}
		result.Packages++
	}
	return result, ctx.Err()
}
