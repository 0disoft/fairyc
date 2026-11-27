package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/0disoft/fairyc/internal/packagesource"
)

func runPackageFetch(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	offline := false
	if len(arguments) == 1 && arguments[0] == "--offline" {
		offline = true
	} else if len(arguments) != 0 {
		return invocationError(stderr, "experimental fetch-packages accepts only optional --offline")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	result, err := (packagesource.Fetcher{}).Fetch(ctx, sourceRoot, offline)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "package fetch failed: %v\n", err); writeErr != nil {
			return ExitInternalFailure
		}
		return ExitPackageFailure
	}
	if _, err := fmt.Fprintf(stdout, "packages fetched: %d package(s), %d downloaded, %d cache hit(s), %d installed, %d reused\n", result.Packages, result.Downloaded, result.CacheHits, result.Installed, result.Reused); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
