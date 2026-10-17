package hir

import (
	"encoding/json"
	"testing"
)

func TestNodeKindsPreserveTextTransport(t *testing.T) {
	if _, err := json.Marshal(StatementKind(255)); err == nil {
		t.Fatal("encoded unknown statement kind")
	}
	if _, err := json.Marshal(ExpressionKind(255)); err == nil {
		t.Fatal("encoded unknown expression kind")
	}
	for k := StatementInvalid; int(k) < len(statementNames); k++ {
		data, err := json.Marshal(k)
		if err != nil {
			t.Fatal(err)
		}
		var got StatementKind
		if err := json.Unmarshal(data, &got); err != nil || got != k {
			t.Fatalf("statement %v: %s, %v", k, data, err)
		}
	}
	for k := ExpressionInvalid; int(k) < len(expressionNames); k++ {
		data, err := json.Marshal(k)
		if err != nil {
			t.Fatal(err)
		}
		var got ExpressionKind
		if err := json.Unmarshal(data, &got); err != nil || got != k {
			t.Fatalf("expression %v: %s, %v", k, data, err)
		}
	}
	data, err := json.Marshal(Statement{Kind: StatementReturn, Expression: &Expression{Kind: ExpressionLiteral}})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["Kind"]) != `"return"` {
		t.Fatalf("Kind = %s", raw["Kind"])
	}
	for _, data := range []string{`"future"`, `1`, `null`} {
		var statement StatementKind
		var expression ExpressionKind
		if json.Unmarshal([]byte(data), &statement) == nil || json.Unmarshal([]byte(data), &expression) == nil {
			t.Fatalf("accepted unknown kind %s", data)
		}
	}
}
