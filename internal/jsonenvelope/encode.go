// Package jsonenvelope joins a fixed object header and an already canonical
// JSON body without serializing or escaping that body a second time.
package jsonenvelope

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrTooLarge = errors.New("JSON envelope exceeds the caller byte limit")

// Marshal appends field as the final member of header. Callers own the fixed
// object header, unique field name and canonical body (from json.Marshal); this
// is not an admission path for untrusted encoded JSON. Neither input is mutated.
// The complete byte size is checked before allocating the output buffer.
func Marshal(header any, field string, body []byte, maxBytes uint64) ([]byte, error) {
	prefix, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if len(prefix) < 2 || prefix[0] != '{' || prefix[len(prefix)-1] != '}' {
		return nil, fmt.Errorf("JSON envelope header must be an object")
	}
	key, err := json.Marshal(field)
	if err != nil {
		return nil, err
	}
	overhead := uint64(len(prefix)) + uint64(len(key)) + 2 // comma, colon, closing brace; replace header brace
	if len(prefix) == 2 {
		overhead-- // no separator before the first field
	}
	limit := min(maxBytes, uint64(^uint(0)>>1))
	if overhead > limit || uint64(len(body)) > limit-overhead {
		return nil, ErrTooLarge
	}
	output := make([]byte, 0, int(overhead)+len(body))
	output = append(output, prefix[:len(prefix)-1]...)
	if len(prefix) > 2 {
		output = append(output, ',')
	}
	output = append(output, key...)
	output = append(output, ':')
	output = append(output, body...)
	return append(output, '}'), nil
}
