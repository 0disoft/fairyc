package adoptionaudit

import (
	"encoding/json"
	"testing"
)

func TestOrdinalObservation(t *testing.T) {
	tests := []struct {
		a, b string
		pass bool
	}{
		{`{"exitCode":0,"stdout":"","stderr":""}`, `{"exitCode":0,"stdout":"","stderr":""}`, true},
		{`{"exitCode":0,"stdout":"한\u0000e\u0301","stderr":"é"}`, `{"exitCode":0,"stdout":"한\u0000e\u0301","stderr":"é"}`, true},
		{`{"exitCode":0,"stdout":"é","stderr":""}`, `{"exitCode":0,"stdout":"e\u0301","stderr":""}`, false},
		{`{"exitCode":0,"stdout":"A","stderr":""}`, `{"exitCode":0,"stdout":"a","stderr":""}`, false},
		{`{"exitCode":0,"stdout":"a\r\n","stderr":""}`, `{"exitCode":0,"stdout":"a\n","stderr":""}`, false},
		{`{"exitCode":1,"stdout":"","stderr":""}`, `{"exitCode":0,"stdout":"","stderr":""}`, false},
		{`{"exitCode":0,"stdout":null,"stderr":""}`, `{"exitCode":0,"stdout":"","stderr":""}`, false},
		{`{"stdout":"","stderr":""}`, `{"exitCode":0,"stdout":"","stderr":""}`, false},
		{`{"exitCode":0,"stdout":"","stderr":"é"}`, `{"exitCode":0,"stdout":"","stderr":"e\u0301"}`, false},
	}
	for i, tt := range tests {
		if (compare(json.RawMessage(tt.a), json.RawMessage(tt.b)) == nil) != tt.pass {
			t.Fatalf("case %d", i)
		}
	}
}
