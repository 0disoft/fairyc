// Observe string decoding through the owning Go encoding/json implementation.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
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
