// Reference tables and checksums only. Runtime path logic is implemented in Fairy.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"unicode"
)

type caseRange struct {
	Lo, Hi uint32
	Delta  int32
}
type special struct{ From, To uint32 }

func canonical(r rune) rune {
	members := []rune{r}
	for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
		members = append(members, next)
	}
	best := rune(unicode.MaxRune + 1)
	for _, member := range members {
		lower := unicode.ToLower(member)
		for _, candidate := range members {
			if lower == candidate && lower < best {
				best = lower
			}
		}
	}
	if best <= unicode.MaxRune {
		return best
	}
	best = r
	for _, member := range members {
		if member < best {
			best = member
		}
	}
	return best
}
func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: path-unicode-reference <report.json>")
	}
	ranges := make([]caseRange, 0, len(unicode.CaseRanges))
	for _, r := range unicode.CaseRanges {
		if r.Delta[unicode.LowerCase] != 0 {
			ranges = append(ranges, caseRange{r.Lo, r.Hi, r.Delta[unicode.LowerCase]})
		}
	}
	differences := map[rune]rune{}
	var lowerHash, foldHash uint64
	for r := rune(0); r <= unicode.MaxRune; r++ {
		lower, fold := unicode.ToLower(r), canonical(r)
		lowerHash = (lowerHash*33 + uint64(lower)) % 16777213
		foldHash = (foldHash*33 + uint64(fold)) % 16777213
		if lower != fold {
			if old, ok := differences[r]; ok && old != fold {
				return fmt.Errorf("inconsistent folded lower rune")
			}
			differences[r] = fold
		}
	}
	specials := make([]special, 0, len(differences))
	for from, to := range differences {
		specials = append(specials, special{uint32(from), uint32(to)})
	}
	sort.Slice(specials, func(i, j int) bool { return specials[i].From < specials[j].From })
	report := struct {
		Schema, UnicodeVersion      string
		Ranges                      []caseRange
		Specials                    []special
		LowerChecksum, FoldChecksum uint64
	}{"fairyc.path-unicode-reference/v1", unicode.Version, ranges, specials, lowerHash, foldHash}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(args[0], append(data, '\n'), 0600); err != nil {
		return err
	}
	fmt.Printf("Unicode %s: %d lower ranges; %d fold exceptions\n", unicode.Version, len(ranges), len(specials))
	return nil
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
