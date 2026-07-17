package source

import "testing"

func TestExpectedModule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{path: "main.fy", want: "main", ok: true},
		{path: "net/http.fy", want: "net::http", ok: true},
		{path: `net\http.fy`, want: "net::http", ok: true},
		{path: "../escape.fy", ok: false},
		{path: "bad-name/main.fy", ok: false},
		{path: "main.txt", ok: false},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			t.Parallel()
			got, err := NewFile(test.path, nil).ExpectedModule()
			if test.ok && err != nil {
				t.Fatalf("ExpectedModule() error = %v", err)
			}
			if !test.ok && err == nil {
				t.Fatalf("ExpectedModule() = %q, want error", got)
			}
			if got != test.want {
				t.Fatalf("ExpectedModule() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBudgetExhaustsEveryCounter(t *testing.T) {
	t.Parallel()
	limits := Limits{
		MaxBytes:       1,
		MaxTokens:      1,
		MaxNodes:       1,
		MaxDepth:       1,
		MaxRepairs:     1,
		MaxDiagnostics: 1,
		MaxWorkUnits:   1,
	}
	if budget := NewBudget(limits, 2); !budget.Exhausted() {
		t.Fatal("oversized source did not exhaust budget")
	}
	checks := []struct {
		name string
		run  func(*Budget) bool
	}{
		{name: "tokens", run: func(b *Budget) bool { b.ChargeToken(); return b.ChargeToken() }},
		{name: "nodes", run: func(b *Budget) bool { b.ChargeNode(); return b.ChargeNode() }},
		{name: "depth", run: func(b *Budget) bool { b.Enter(); return b.Enter() }},
		{name: "repairs", run: func(b *Budget) bool { b.ChargeRepair(); return b.ChargeRepair() }},
		{name: "diagnostics", run: func(b *Budget) bool { b.ChargeDiagnostic(); return b.ChargeDiagnostic() }},
		{name: "work", run: func(b *Budget) bool { b.ChargeWork(1); return b.ChargeWork(1) }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			budget := NewBudget(limits, 1)
			if check.run(budget) || !budget.Exhausted() {
				t.Fatal("counter did not fail closed at its boundary")
			}
		})
	}
}
