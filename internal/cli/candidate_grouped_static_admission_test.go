package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func groupedStaticAdmissionMembers(group, prefix string, count int) string {
	var out strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&out, "@interface(%s) pub fn T %s%d<T>(T value){}\r\n", group, prefix, i)
	}
	return out.String()
}

func groupedStaticAdmissionProviders(prefix, scalar string, count int) string {
	var out strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&out, "@implements(%s%d,%s) fn %s %s_%s_%d(%s value){return value;}\r\n", prefix, i, scalar, scalar, prefix, scalar, i, scalar)
	}
	return out.String()
}

func TestCandidateGroupedStaticAdmission(t *testing.T) {
	const module = "module sample; // 한글\r\n"
	const main = "fn i32 main(){return 0;}"
	two := groupedStaticAdmissionMembers("G", "op", 2)
	providers := groupedStaticAdmissionProviders("op", "i32", 2)
	var allScalars strings.Builder
	for _, scalar := range []string{"i32", "i64", "u8", "u32", "usize", "bool", "i8", "i16", "isize", "u16", "u64", "f32", "f64"} {
		allScalars.WriteString(groupedStaticAdmissionProviders("op", scalar, 2))
	}
	cases := []struct {
		name, body string
		accepted   bool
	}{
		{"empty-group", two, true},
		{"complete", two + providers, true},
		{"thirteen-scalars", two + allScalars.String(), true},
		{"two-labels", two + providers + groupedStaticAdmissionMembers("H", "other", 2) + groupedStaticAdmissionProviders("other", "i32", 2), true},
		{"sixteen", groupedStaticAdmissionMembers("G", "op", 16) + groupedStaticAdmissionProviders("op", "i32", 16), true},
		{"label64", groupedStaticAdmissionMembers(strings.Repeat("g", 64), "op", 2) + providers, true},
		{"singleton", groupedStaticAdmissionMembers("G", "op", 1), false},
		{"seventeen", groupedStaticAdmissionMembers("G", "op", 17), false},
		{"size-priority", groupedStaticAdmissionMembers("Early", "first", 1) + groupedStaticAdmissionMembers("Late", "last", 17), false},
		{"label65", groupedStaticAdmissionMembers(strings.Repeat("g", 65), "op", 2), false},
		{"numeric-label", groupedStaticAdmissionMembers("1", "op", 2), false},
		{"quoted-label", groupedStaticAdmissionMembers(`"G"`, "op", 2), false},
		{"qualified-label", groupedStaticAdmissionMembers("G::H", "op", 2), false},
		{"extra-label", groupedStaticAdmissionMembers("G,H", "op", 2), false},
		{"private-member", strings.Replace(two, "pub fn", "fn", 1), false},
		{"partial-unused", two + groupedStaticAdmissionProviders("op", "i32", 1), false},
		{"partial-unused-scalar", two + providers + groupedStaticAdmissionProviders("op", "u8", 1), false},
		{"two-partials-priority", two + groupedStaticAdmissionMembers("H", "other", 2) + groupedStaticAdmissionProviders("other", "u8", 1) + groupedStaticAdmissionProviders("op", "i32", 1), false},
		{"duplicate", two + providers + strings.Replace(groupedStaticAdmissionProviders("op", "i32", 1), "op_i32_0", "duplicate", 1), false},
		{"wrong-provider-result", two + strings.Replace(providers, "fn i32 op_i32_0", "fn i64 op_i32_0", 1), false},
		{"group-label-provider", two + "@implements(G,i32) fn i32 wrong(i32 value){return value;}", false},
		{"operation-before-bad-provider", two + "@implements(op0,i32) fn i64 wrong(i32 value){return 0;}@interface(H) pub fn i32 broken<T>(T value){}", false},
		{"operation-before-duplicate", two + providers + "@implements(op0,i32) fn i32 duplicate(i32 value){return value;}@interface(H) pub fn i32 broken<T>(T value){}", false},
		{"legacy-single", "@interface fn T transform<T>(T value){}@implements(transform,i32) fn i32 one(i32 value){return value;}", true},
	}
	var helpers, driver strings.Builder
	for i, test := range cases {
		input := module + test.body + main
		parsed, err := frontend.AnalyzeSyntax(source.NewFile("sample.fy", []byte(input)), source.DefaultLimits())
		if err != nil || parsed.AST == nil {
			t.Fatalf("%s parser: %v", test.name, err)
		}
		checked, err := compiler.CheckExplicitFiles([]semantic.File{{Path: "sample.fy", Module: parsed.AST}})
		if err != nil || checked.Verified != test.accepted {
			t.Fatalf("%s Go: %v verified=%t diagnostics=%+v unsupported=%+v", test.name, err, checked.Verified, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
		}
		fmt.Fprintf(&helpers, "fn i32 group_case_%d(){let p=switch(make(b%s)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};var usize work=0;let problem=operations::validate_problem(ref p,ref mut work);", i, strconv.Quote(input))
		if test.accepted {
			helpers.WriteString("assert(problem.code==0 && problem.declaration==0);")
		} else {
			found := false
			for _, diagnostic := range checked.Semantic.Diagnostics {
				if diagnostic.Code == "E4501" {
					found = true
					fmt.Fprintf(&helpers, "assert(problem.code==4501 && problem.declaration!=0);let d=syntax::declaration_at(ref p,problem.declaration-1);assert(d.start==%d && d.end==%d);", diagnostic.Primary.StartByte, diagnostic.Primary.EndByte)
					break
				}
			}
			if !found {
				t.Fatalf("%s missing E4501 oracle: %+v", test.name, checked.Semantic.Diagnostics)
			}
		}
		helpers.WriteString("return 0;}\n")
		fmt.Fprintf(&driver, "assert(group_case_%d()==0);", i)
	}
	t.Logf("%d source controls; exact Go/Fairy error byte spans with UTF-8 and CRLF", len(cases))
	candidateInlineProbe(t, []string{"static_operation_declarations"}, "import static_operation_declarations as operations;\n"+helpers.String(), driver.String())
}
