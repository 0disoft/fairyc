package interpreter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func regionCopyInput(backing string, offset, length uint64) runtimeValue {
	view := &indexedViewState{offset: offset, length: length}
	switch backing {
	case "static":
		view.staticData = "a\x00\xffz"
	case "packed":
		view.array = &arrayState{byteElements: []byte{'a', 0, 255, 'z'}}
	case "typed":
		view.array = &arrayState{}
		for _, value := range []byte{'a', 0, 255, 'z'} {
			view.array.elements = append(view.array.elements, runtimeValue{kind: valueU8, typ: hir.TypeU8, scalar: uint64(value)})
		}
	}
	return runtimeValue{kind: valueView, typ: hir.SharedBorrow(hir.TypeByteSlice), payload: view}
}

func TestRegionCopyBackingAndRangeBoundaries(t *testing.T) {
	for _, backing := range []string{"static", "packed", "typed"} {
		for _, test := range []struct {
			offset, length uint64
			valid          bool
		}{
			{0, 4, true}, {1, 3, true}, {3, 1, true}, {4, 0, true},
			{4, 1, false}, {3, 2, false}, {5, 1, false}, {^uint64(0), 2, false},
		} {
			t.Run(fmt.Sprintf("%s/%d/%d", backing, test.offset, test.length), func(t *testing.T) {
				input := regionCopyInput(backing, test.offset, test.length)
				r := executionState{steps: 8}
				got, err := r.regionBytes(input, true)
				if (err == nil) != test.valid || r.steps != 8-int(test.length) {
					t.Fatalf("copy validity/budget: value=%+v steps=%d error=%v", got, r.steps, err)
				}
				if test.valid {
					want := []byte{'a', 0, 255, 'z'}[test.offset : test.offset+test.length]
					if !bytes.Equal(got.view().array.byteElements, want) {
						t.Fatalf("got %v, want %v", got.view().array.byteElements, want)
					}
				} else {
					want := "byte-view read exceeds its array backing"
					if backing == "static" {
						want = "byte-view read exceeds its static backing"
					}
					if err.Error() != want {
						t.Fatalf("error=%v, want %q", err, want)
					}
				}
			})
		}
	}
}

func TestRegionCopyPreservesTypedElementChecksAndGuards(t *testing.T) {
	for _, invalid := range []runtimeValue{i32Value(255), {kind: valueU8, typ: hir.TypeU8, scalar: 256}} {
		input := regionCopyInput("typed", 0, 4)
		input.view().array.elements[2] = invalid
		r := executionState{steps: 4}
		if _, err := r.regionBytes(input, true); err == nil || err.Error() != "byte-view array backing contains a non-u8 value" {
			t.Fatalf("typed backing validation skipped: %v", err)
		}
	}
	for _, backing := range []string{"static", "packed", "typed"} {
		input := regionCopyInput(backing, 0, 4)
		r := executionState{steps: 3}
		var capability CapabilityError
		if _, err := r.regionBytes(input, true); !errors.As(err, &capability) || capability.Kind != LimitStep || r.steps != 3 {
			t.Fatalf("copy exceeded step budget: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r.context, r.steps = ctx, 4
		if _, err := r.regionBytes(input, true); !errors.Is(err, context.Canceled) || r.steps != 4 {
			t.Fatalf("copy ignored cancellation: %v", err)
		}
	}
}

func TestRegionCopyEmptyViewReadsNoBacking(t *testing.T) {
	for _, backing := range []string{"static", "packed", "typed"} {
		// No bytes are read for an empty view, even if its offset is unusable.
		input := regionCopyInput(backing, ^uint64(0), 0)
		r := executionState{}
		got, err := r.regionBytes(input, true)
		if err != nil || got.view().length != 0 || r.steps != 0 {
			t.Fatalf("empty %s copy touched backing: %v", backing, err)
		}
	}
}
