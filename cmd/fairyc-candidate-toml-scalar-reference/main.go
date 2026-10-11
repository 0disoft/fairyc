// Test-only scalar syntax oracle; the compiler implementation is Fairy.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

func run(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("request, binary and report paths required")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return fmt.Errorf("request exceeds 2 MiB")
	}
	var request struct {
		Schema string `json:"schema"`
		Cases  []struct {
			Name string `json:"name"`
			Raw  []byte `json:"raw"`
		} `json:"cases"`
	}
	if err = json.Unmarshal(data, &request); err != nil {
		return err
	}
	if request.Schema != "fairyc.candidate-toml-scalars-request/v1" || len(request.Cases) == 0 || len(request.Cases) > 2048 {
		return fmt.Errorf("invalid request")
	}
	var output bytes.Buffer
	output.WriteString("FYATOM01\n")
	seen := map[string]bool{}
	rows := make([]struct {
		Name string `json:"name"`
		Kind uint32 `json:"kind"`
	}, 0, len(request.Cases))
	for _, item := range request.Cases {
		if item.Name == "" || seen[item.Name] || len(item.Raw) > 4096 {
			return fmt.Errorf("invalid case %q", item.Name)
		}
		seen[item.Name] = true
		var value map[string]any
		metadata, parseErr := toml.Decode("value="+string(item.Raw)+"\n", &value)
		var kind uint32
		if parseErr == nil && len(value) == 1 && len(metadata.Keys()) == 1 {
			switch metadata.Type("value") {
			case "Integer":
				kind = 6
			case "Float":
				kind = 7
			case "Datetime":
				kind = 8
			}
		}
		if err = binary.Write(&output, binary.LittleEndian, uint16(len(item.Raw))); err != nil {
			return err
		}
		output.WriteByte(byte(kind))
		output.Write(item.Raw)
		rows = append(rows, struct {
			Name string `json:"name"`
			Kind uint32 `json:"kind"`
		}{item.Name, kind})
	}
	if output.Len() > 65536 {
		return fmt.Errorf("fixture exceeds 64 KiB")
	}
	if err = os.WriteFile(args[1], output.Bytes(), 0600); err != nil {
		return err
	}
	report := struct {
		Schema string `json:"schema"`
		Cases  any    `json:"cases"`
	}{"fairyc.candidate-toml-scalars-reference/v1", rows}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(args[2], append(encoded, '\n'), 0600)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
