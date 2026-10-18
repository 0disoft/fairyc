package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/adoptionaudit"
)

func main() {
	var o adoptionaudit.Options
	flag.StringVar(&o.Kind, "kind", "", "ledger, log, pack or kv")
	flag.StringVar(&o.Receipt, "receipt", "", "retained receipt.json (required)")
	flag.StringVar(&o.Compiler, "compiler", "", "retained compiler to hash, never execute (required)")
	flag.StringVar(&o.PathMapFrom, "path-map-from", "", "old absolute evidence root when relocating")
	flag.StringVar(&o.PathMapTo, "path-map-to", "", "new absolute evidence root when relocating")
	flag.Parse()
	if flag.NArg() != 0 || o.Kind == "" || o.Receipt == "" || o.Compiler == "" || (o.PathMapFrom == "") != (o.PathMapTo == "") {
		fmt.Fprintln(os.Stderr, "provide -kind, -receipt and -compiler; path mapping requires both roots")
		os.Exit(2)
	}
	result, err := adoptionaudit.Audit(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
