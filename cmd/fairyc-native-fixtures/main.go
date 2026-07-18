// Command fairyc-native-fixtures runs the bounded Linux C differential fixture lane.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/cbackendnative"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: fairyc-native-fixtures <c-emission-manifest>")
		os.Exit(2)
	}
	report, err := cbackendnative.Run(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "native C differential failed: %v\n", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode native C report: %v\n", err)
		os.Exit(1)
	}
}
