// Test-only Go oracle for the Fairy candidate's canonical token stream.
// It never participates in candidate parsing, compilation or native builds.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

type input struct {
	Name     string         `json:"name"`
	Source   []byte         `json:"source"`
	Previous bool           `json:"previous"`
	Limits   *source.Limits `json:"limits,omitempty"`
}

type request struct {
	Schema string  `json:"schema"`
	Cases  []input `json:"cases"`
}

type observation struct {
	Name   string `json:"name"`
	Stdout []byte `json:"stdout"`
}

func run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("request and reference report paths are required")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return fmt.Errorf("reference request exceeds 2 MiB")
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	if req.Schema != "fairyc.candidate-lexer-reference-request/v1" || len(req.Cases) == 0 || len(req.Cases) > 2048 {
		return fmt.Errorf("invalid reference schema or case count")
	}
	seen := map[string]bool{}
	rows := make([]observation, 0, len(req.Cases))
	for _, item := range req.Cases {
		if item.Name == "" || seen[item.Name] || len(item.Source) > 32768 {
			return fmt.Errorf("invalid or duplicate reference case %q", item.Name)
		}
		seen[item.Name] = true
		limits := source.DefaultLimits()
		limits.MaxBytes = 32768
		if item.Limits != nil {
			limits = *item.Limits
		}
		budget := source.NewBudget(limits, len(item.Source))
		file := source.NewFile("input.fy", item.Source)
		lex := lexer.Lex
		if item.Previous {
			lex = lexer.LexPreviousRevision
		}
		result := lex(file, budget)
		var out bytes.Buffer
		for _, token := range result.Tokens {
			fmt.Fprintf(&out, "%d %d %d %d %d ", token.Kind, token.Trivia, token.Span.StartByte, token.Span.EndByte, len(token.Cooked))
			for _, b := range []byte(token.Cooked) {
				fmt.Fprintf(&out, "%d ", b)
			}
		}
		fmt.Fprintf(&out, "%d ", len(result.Diagnostics))
		for _, d := range result.Diagnostics {
			for _, value := range []string{d.Code, string(d.Repair), d.Token} {
				fmt.Fprintf(&out, "%d ", len(value))
				for _, b := range []byte(value) {
					fmt.Fprintf(&out, "%d ", b)
				}
			}
			fmt.Fprintf(&out, "%d %d ", d.Span.StartByte, d.Span.EndByte)
		}
		rows = append(rows, observation{Name: item.Name, Stdout: out.Bytes()})
	}
	report := struct {
		Schema string        `json:"schema"`
		Cases  []observation `json:"cases"`
	}{"fairyc.candidate-lexer-reference/v1", rows}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(args[1], append(encoded, '\n'), 0o600)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
