// Windows standard-library oracle only; command encoding remains Fairy code.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) != 2 {
		panic("usage: candidate-command-reference OUTPUT")
	}
	type row struct {
		program   string
		arguments []string
	}
	var rows []row
	values := []string{"", "plain", "a b", "a\tb", `x\"y\`, `ends\`, `two\\"`, "한글🧚", "a\nb", `$()&<>|;`, ` with slash\`, `"quoted"`}
	for _, program := range []string{`fairyc.exe`, `C:\tool dir\fairyc.exe`, `C:\한글\fairyc.exe`, `C:\tool dir\`, "tab\tname.exe"} {
		rows = append(rows, row{program, nil})
		for _, value := range values {
			rows = append(rows, row{program, []string{value}})
		}
		rows = append(rows, row{program, values})
	}
	var out bytes.Buffer
	out.WriteString("FWCMD1\n")
	putNumber := func(n int) {
		if err := binary.Write(&out, binary.LittleEndian, uint32(n)); err != nil {
			panic(err)
		}
	}
	put := func(data []byte) { putNumber(len(data)); out.Write(data) }
	putNumber(len(rows))
	for _, r := range rows {
		put([]byte(r.program))
		putNumber(len(r.arguments))
		parts := []string{syscall.EscapeArg(r.program)}
		for _, arg := range r.arguments {
			put([]byte(arg))
			parts = append(parts, syscall.EscapeArg(arg))
		}
		wide, err := syscall.UTF16FromString(strings.Join(parts, " "))
		if err != nil {
			panic(err)
		}
		var data bytes.Buffer
		for _, word := range wide {
			if err := binary.Write(&data, binary.LittleEndian, word); err != nil {
				panic(err)
			}
		}
		put(data.Bytes())
	}
	if err := os.WriteFile(os.Args[1], out.Bytes(), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("%d exact Go command cases\n", len(rows))
}
