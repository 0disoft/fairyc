// Test child and live-process oracle only; process orchestration is Fairy code.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	switch os.Args[1] {
	case "stdio":
		data, err := io.ReadAll(os.Stdin)
		if err != nil || !bytes.Equal(data, []byte("in\x00put\n")) {
			fmt.Fprintf(os.Stderr, "wrong stdin: %q (%v)", data, err)
			os.Exit(2)
		}
		fmt.Fprint(os.Stdout, "child stdout\n")
		fmt.Fprint(os.Stderr, "child stderr\n")
		os.Exit(41)
	case "args":
		want := []string{"args", "", "a b", "x\\\"y\\", "한글🧚", "--flag"}
		if !reflect.DeepEqual(os.Args[1:], want) {
			fmt.Fprintf(os.Stderr, "wrong argv: %q", os.Args[1:])
			os.Exit(2)
		}
		os.Exit(37)
	case "exit":
		n, err := strconv.ParseUint(os.Args[2], 10, 32)
		if err != nil {
			panic(err)
		}
		os.Exit(int(n))
	case "sleep":
		time.Sleep(time.Minute)
	case "tree":
		self, err := os.Executable()
		if err != nil {
			panic(err)
		}
		child := exec.Command(self, "sleep")
		child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := child.Start(); err != nil {
			panic(err)
		}
		handle, err := syscall.OpenProcess(0x101000, false, uint32(child.Process.Pid))
		if err != nil {
			panic(err)
		}
		defer syscall.CloseHandle(handle)
		var code uint32
		if err := syscall.GetExitCodeProcess(handle, &code); err != nil || code != 259 {
			panic("child was not actually live")
		}
		if err := os.WriteFile(os.Args[2], []byte(fmt.Sprintf("%d %d", child.Process.Pid, code)), 0600); err != nil {
			panic(err)
		}
		if err := child.Wait(); err != nil {
			os.Exit(9)
		}
	default:
		os.Exit(2)
	}
}
