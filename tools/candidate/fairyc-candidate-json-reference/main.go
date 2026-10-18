// Observe string decoding through the owning Go encoding/json implementation.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/lockfile"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "tree" {
		writeTree(os.Args[2])
		return
	}
	if len(os.Args) != 2 {
		panic("usage: fairyc-candidate-json-reference OUTPUT")
	}
	inputs := [][]byte{
		[]byte(`""`), []byte(`"ordinary <>& / text"`),
		[]byte(`"\"\\\/\b\f\n\r\t\u0000"`), []byte(`"한글🙂"`),
		[]byte(`"\uD55c\uAE00\ud83d\ude42"`), []byte(`"\u007F\u0080\u07ff\u0800\uffff"`),
		[]byte(`"\ud800"`), []byte(`"\udfff"`), []byte(`"\ud800A"`),
		[]byte(`"\ud800\ud800\udc00"`), []byte(`"\udbff\udfff"`),
		{'"', 0xff, 0xc0, 0xaf, '"'}, {'"', 0xed, 0xa0, 0x80, '"'},
		{'"', 0xf0, 0x9f, '"'},
		[]byte(`"\x20"`), []byte(`"\uZZZZ"`), []byte(`"\u123"`),
		[]byte(`"\ud800\uZZZZ"`), []byte(`"\ud800\u123"`),
		[]byte(`"unterminated`), []byte{'"', 'a', '\\'},
		{'"', 0, '"'}, {'"', '\n', '"'}, []byte(`null`),
	}
	var output bytes.Buffer
	output.WriteString("FYJDEC1\n")
	writeNumber := func(value int) {
		var data [8]byte
		binary.BigEndian.PutUint64(data[:], uint64(value))
		output.Write(data[:])
	}
	writeBytes := func(value []byte) { writeNumber(len(value)); output.Write(value) }
	writeNumber(len(inputs))
	accepted := 0
	for _, input := range inputs {
		var value string
		err := json.Unmarshal(input, &value)
		// JSON null leaves a Go string unchanged; the string-only lexer rejects it.
		valid := err == nil && len(input) > 0 && input[0] == '"'
		writeBytes(input)
		if valid {
			writeNumber(1)
			accepted++
		} else {
			writeNumber(0)
			value = ""
		}
		writeBytes([]byte(value))
	}
	path := os.Args[1]
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, output.Bytes(), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("JSON strings: %d accepted, %d rejected\n", accepted, len(inputs)-accepted)
}

func writeTree(path string) {
	inputs := [][]byte{
		[]byte(`"text"`), []byte(`0`), []byte(`-0`), []byte(`-1.25e+09`), []byte(`1E-09`),
		[]byte(`true`), []byte(`false`), []byte(`null`), []byte(`{}`), []byte(`[]`),
		[]byte(" \r\n { \"a\" : [true, {\"b\":null}], \"c\":{} } \t"),
		[]byte(`{"a":{"a":0},"b":{"a":1}}`), []byte(`[{"a":0},{"a":1}]`),
		[]byte(`{"a":0,"A":1}`), []byte(`{"b":0,"a":1}`),
		[]byte(`{"a":0,"a":1}`), []byte(`{"a":0,"\u0061":1}`),
		[]byte(`{"a":{"a":0,"a":1}}`), []byte(`{"a":{"b":0},"a":1}`),
		[]byte(`{"\ud800":0,"\ufffd":1}`), []byte{'{', '"', 0xff, '"', ':', '0', ',', '"', '\\', 'u', 'f', 'f', 'f', 'd', '"', ':', '1', '}'},
		[]byte(``), []byte(` `), []byte(`{} []`), []byte(`true false`), []byte(`{}x`),
		[]byte(`{`), []byte(`[`), []byte(`{]`), []byte(`[}`), []byte(`[,]`),
		[]byte(`[1,]`), []byte(`{"a":0,}`), []byte(`{"a" 0}`), []byte(`{"a":}`),
		[]byte(`{1:0}`), []byte(`{a:0}`), []byte(`{"a":0 "b":1}`), []byte(`[1 2]`),
		[]byte(`[01]`), []byte(`[1.]`), []byte(`[1e+]`), []byte(`{"a":"\x"}`),
		[]byte(`{"a":0}null`), []byte(`{"a":0,,"b":1}`),
	}
	var output bytes.Buffer
	output.WriteString("FYJTREE1")
	writeNumber := func(value int) {
		var data [8]byte
		binary.BigEndian.PutUint64(data[:], uint64(value))
		output.Write(data[:])
	}
	writeNumber(len(inputs))
	counts := map[int]int{}
	for _, input := range inputs {
		code := 0
		if !json.Valid(input) {
			code = 1
		} else if _, err := lockfile.Parse(input); err != nil && strings.Contains(err.Error(), "duplicate JSON key") {
			// The owning lock parser performs its exact decoded-key scan before
			// schema validation, including for these generic JSON observations.
			code = 4
		}
		writeNumber(len(input))
		output.Write(input)
		writeNumber(code)
		counts[code]++
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, output.Bytes(), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("JSON tree: %d accepted, %d syntax, %d duplicate\n", counts[0], counts[1], counts[4])
}
