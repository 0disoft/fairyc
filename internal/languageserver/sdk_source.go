package languageserver

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

const sdkSourcePrefix = "fairyc-sdk://stdlib/"

func sdkSourceURI(relative string, text []byte) string {
	return fmt.Sprintf("%s%x/%s", sdkSourcePrefix, sha256.Sum256(text), relative)
}

func readSDKSource(uri string) (string, bool) {
	suffix, ok := strings.CutPrefix(uri, sdkSourcePrefix)
	if !ok || len(suffix) > 256 {
		return "", false
	}
	digest, relative, ok := strings.Cut(suffix, "/")
	if !ok || len(digest) != 2*sha256.Size {
		return "", false
	}
	text, err := sdkstdlib.ReadSource(relative)
	if err != nil || sdkSourceURI(relative, text) != uri {
		return "", false
	}
	return string(text), true
}
