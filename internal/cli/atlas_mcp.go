package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/0disoft/fairyc/internal/atlasmcp"
	"github.com/0disoft/fairyc/internal/symbolidentity"
)

func runAtlasMCP(arguments []string, root string, stdin io.Reader, stdout, stderr io.Writer) int {
	o := impactOptions{}
	messages := 0
	values := map[string]*int{"--max-packages": &o.packages, "--max-calls": &o.calls, "--max-depth": &o.depth, "--max-symbols": &o.symbols, "--max-output-bytes": &o.output, "--max-requests": &messages}
	flags := []flagDefinition{{Name: "--allow-impact", Kind: flagSwitch}}
	required := []string{"--allow-impact", "--max-packages", "--max-calls", "--max-depth", "--max-symbols", "--max-output-bytes", "--max-requests"}
	for _, flag := range required[1:] {
		flags = append(flags, flagDefinition{Name: flag, Kind: flagScalar})
	}
	_, err := parseCommandFlags(arguments, flagSchema{Command: "atlas-mcp", Experimental: true, Flags: flags, Required: required}, func(flag, value string) error {
		if flag == "--allow-impact" {
			return nil
		}
		n, err := strconv.Atoi(value)
		max := 65536
		min := 1
		switch flag {
		case "--max-depth":
			max = 64
			min = 0
		case "--max-symbols":
			max = 4096
		case "--max-output-bytes":
			max = 4 << 20
			min = 2048
		case "--max-requests":
			max = 1024
		}
		if err != nil || strconv.Itoa(n) != value || n < min || n > max {
			return fmt.Errorf("invalid %s", flag)
		}
		*values[flag] = n
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	session := atlasImpactSession{root: root}
	err = atlasmcp.Serve(stdin, stdout, messages, o.output, func(uid, symbol string) (json.RawMessage, error) {
		if err := validateAtlasPackageUID("atlas-mcp", uid); err != nil {
			return nil, err
		}
		if _, err := symbolidentity.ParseVisibleID(symbol); err != nil {
			if _, err := symbolidentity.ParseFullDigest(symbol); err != nil {
				return nil, err
			}
		}
		selected := o
		selected.uid, selected.symbol = uid, symbol
		selected.output -= 1024
		data, code := session.document(selected, io.Discard)
		if code != ExitSuccess {
			return nil, fmt.Errorf("query unavailable")
		}
		return data, nil
	})
	if err != nil {
		if errors.Is(err, atlasmcp.ErrOutput) {
			return outputError(stderr)
		}
		return invocationError(stderr, "MCP session failed its framing, I/O or resource policy")
	}
	return ExitSuccess
}
