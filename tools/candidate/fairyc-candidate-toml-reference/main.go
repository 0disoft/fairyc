// Test-only string oracle; compiler string decoding is implemented in Fairy.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type input struct {
	Name   string `json:"name"`
	Source []byte `json:"source"`
	Key    bool   `json:"key"`
}
type observation struct {
	Name     string `json:"name"`
	Accepted bool   `json:"accepted"`
	Decoded  []byte `json:"decoded"`
}

func observe(item input) observation {
	row := observation{Name: item.Name}
	if item.Key {
		var value map[string]int
		metadata, err := toml.Decode(string(item.Source)+" = 1\n", &value)
		if err == nil && len(metadata.Keys()) == 1 && len(metadata.Keys()[0]) == 1 && len(value) == 1 {
			row.Accepted = true
			row.Decoded = []byte(metadata.Keys()[0][0])
		}
	} else {
		var value map[string]string
		metadata, err := toml.Decode("value = "+string(item.Source)+"\n", &value)
		if err == nil && len(metadata.Keys()) == 1 && metadata.Type("value") == "String" {
			row.Accepted = true
			row.Decoded = []byte(value["value"])
		}
	}
	return row
}

func run(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("request, binary fixture and report paths are required")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return fmt.Errorf("request exceeds 2 MiB")
	}
	var request struct {
		Schema string  `json:"schema"`
		Cases  []input `json:"cases"`
	}
	if err = json.Unmarshal(data, &request); err != nil {
		return err
	}
	if request.Schema != "fairyc.candidate-toml-strings-request/v1" || len(request.Cases) == 0 || len(request.Cases) > 2048 {
		return fmt.Errorf("invalid request schema or count")
	}
	seen := map[string]bool{}
	var rows []observation
	var output bytes.Buffer
	output.WriteString("FYTSTR01\n")
	for _, item := range request.Cases {
		if item.Name == "" || seen[item.Name] || len(item.Source) > 4096 {
			return fmt.Errorf("invalid or duplicate case %q", item.Name)
		}
		seen[item.Name] = true
		row := observe(item)
		if len(row.Decoded) > 4096 {
			return fmt.Errorf("decoded fixture exceeds 4096 bytes")
		}
		mode, accepted := byte(0), byte(0)
		if item.Key {
			mode = 1
		}
		if row.Accepted {
			accepted = 1
		}
		header := [6]byte{mode, accepted}
		binary.LittleEndian.PutUint16(header[2:4], uint16(len(item.Source)))
		binary.LittleEndian.PutUint16(header[4:6], uint16(len(row.Decoded)))
		output.Write(header[:])
		output.Write(item.Source)
		output.Write(row.Decoded)
		rows = append(rows, row)
	}
	if output.Len() > 65536 {
		return fmt.Errorf("binary fixture exceeds 65536 bytes")
	}
	report := struct {
		Schema string        `json:"schema"`
		Cases  []observation `json:"cases"`
	}{"fairyc.candidate-toml-strings-reference/v1", rows}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(args[1], output.Bytes(), 0600); err != nil {
		return err
	}
	return os.WriteFile(args[2], append(encoded, '\n'), 0600)
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
