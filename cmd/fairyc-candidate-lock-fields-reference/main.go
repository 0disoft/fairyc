// Observe closed lockfield decoding; semantic validation belongs to a later unit.
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

func observation(d lockfile.Document) []byte {
	var out bytes.Buffer
	number := func(value int) {
		var data [8]byte
		binary.BigEndian.PutUint64(data[:], uint64(value))
		out.Write(data[:])
	}
	text := func(value string) { number(len(value)); out.WriteString(value) }
	flag := func(value bool) {
		if value {
			number(1)
		} else {
			number(0)
		}
	}
	for _, value := range []string{d.Schema, d.PackageInstanceAlgorithm, d.SourceDigestAlgorithm, d.Language, d.Root} {
		text(value)
	}
	flag(d.Packages == nil)
	number(len(d.Packages))
	for _, p := range d.Packages {
		for _, value := range []string{p.ID, p.Role, p.PackageUID, p.SourceKind, p.LocalPath, p.SDKIdentity, p.RuntimeABI, p.SourceIdentity, p.SourceDigest, p.Version} {
			text(value)
		}
		flag(p.Features == nil)
		number(len(p.Features))
		for _, feature := range p.Features {
			text(feature)
		}
	}
	flag(d.Dependencies == nil)
	number(len(d.Dependencies))
	for _, e := range d.Dependencies {
		text(e.Owner)
		text(e.Binding)
		text(e.Target)
	}
	return out.Bytes()
}

func main() {
	if len(os.Args) != 3 {
		panic("usage: fairyc-candidate-lock-fields-reference GO_LOCK_FIXTURES OUTPUT")
	}
	var inputs [][]byte
	for _, name := range []string{"None", "CandidateUnused", "Core", "AliasCore", "Vector", "ChildOnly", "Both", "ChildVector", "UnreachableVector", "FeatureSplit", "InstanceAliases", "LongTrivia", "SimilarNamespaces"} {
		input, err := os.ReadFile(filepath.Join(os.Args[1], name, "lock.json"))
		if err != nil {
			panic(err)
		}
		inputs = append(inputs, input)
		if name == "None" || name == "Core" || name == "FeatureSplit" {
			var fields map[string]any
			if err := json.Unmarshal(input, &fields); err != nil {
				panic(err)
			}
			// Map encoding changes key order and indentation; the decoder must
			// retain the same values rather than require canonical output bytes.
			reordered, err := json.MarshalIndent(fields, "", "    ")
			if err != nil {
				panic(err)
			}
			inputs = append(inputs, reordered)
		}
	}
	for _, input := range []string{
		`{}`, `null`, `{"schema":null,"packages":null,"dependencies":null}`,
		`{"schema":"first","SCHEMA":"last"}`, `{"schema":"first","SCHEMA":null}`,
		`{"ſchema":"folded","PACKAGES":[{"pacKageUid":"folded","features":[]}]}`,
		`{"pack\u0061ges":[{"id":"escaped","features":[]}]}`,
		`{"packages":[null],"dependencies":[null]}`,
		`{"packages":[{"id":"first","features":[]}],"PACKAGES":[{"id":"last","FEATURES":["x",null]}]}`,
		`{"packages":[{"features":["x"],"FEATURES":null}]}`,
		`{"packages":[{"features":["old"],"FEATURES":["new"]}],"dependencies":[]}`,
		`{"packages":[{"id":"first","ID":null,"features":[]}]}`,
		`{"dependencies":[{"owner":"a","OWNER":"b","binding":null,"target":"c"}]}`,
		`{"dependencies":[],"DEPENDENCIES":null}`, `{"packages":[],"PACKAGES":null}`,
		`{"packages":[{"id":"first","features":["f"]}],"PACKAGES":[{"role":"dependency"}]}`,
		`{"packages":[{"features":["x"],"FEATURES":[null]}]}`,
		`{"packages":[{"id":"one"},{"id":"two"}],"PACKAGES":[{}],"Packages":[{},{}]}`,
		`{"packages":[{"features":["x","y"],"FEATURES":["z"],"Features":[null,null]}]}`,
		`{"dependencies":[{"owner":"one"},{"owner":"two"}],"DEPENDENCIES":[{}],"Dependencies":[null,{}]}`,
		`{"packages":[{"id":"old"}],"PACKAGES":[],"Packages":[{}]}`,
		`{"packages":[{"features":["old"],"FEATURES":[],"Features":[null]}]}`,
		`{"packages":[{"id":"old"}],"PACKAGES":null,"Packages":[{}]}`,
		`{"wat":0}`, `{"packages":[{"wat":null}]}`, `{"dependencies":[{"wat":null}]}`,
		`{"schema":0}`, `{"schema":true}`, `{"packages":{}}`, `{"packages":[true]}`,
		`{"packages":[{"features":{}}]}`, `{"packages":[{"features":[false]}]}`,
		`{"dependencies":{}}`, `{"dependencies":[[]]}`, `{"dependencies":[{"owner":1}]}`,
		`{"schema":"x","schema":"y"}`, `{"packages":[{"features":[],"features":[]}]}`,
		`{"dependencies":[{"owner":"x","own\u0065r":"x"}]}`, `{} []`, `{"schema":}`,
	} {
		inputs = append(inputs, []byte(input))
	}
	var output bytes.Buffer
	output.WriteString("FYLFIELD")
	number := func(value int) {
		var data [8]byte
		binary.BigEndian.PutUint64(data[:], uint64(value))
		output.Write(data[:])
	}
	blob := func(value []byte) { number(len(value)); output.Write(value) }
	number(len(inputs))
	counts := map[int]int{}
	for _, input := range inputs {
		code := 0
		var document lockfile.Document
		if !json.Valid(input) {
			code = 1
		} else if _, err := lockfile.Parse(input); err != nil && strings.Contains(err.Error(), "duplicate JSON key") {
			code = 4
		} else {
			decoder := json.NewDecoder(bytes.NewReader(input))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&document); err != nil {
				if strings.HasPrefix(err.Error(), "json: unknown field") {
					code = 5
				} else {
					code = 6
				}
			}
		}
		blob(input)
		number(code)
		if code == 0 {
			blob(observation(document))
		} else {
			blob(nil)
		}
		counts[code]++
	}
	if output.Len() > 65536 {
		panic("reference fixture exceeds the bounded test reader")
	}
	path := os.Args[2]
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, output.Bytes(), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("Lock fields: %d cases, %d accepted, %d syntax, %d duplicate, %d unknown, %d type; %d bytes\n", len(inputs), counts[0], counts[1], counts[4], counts[5], counts[6], output.Len())
}
