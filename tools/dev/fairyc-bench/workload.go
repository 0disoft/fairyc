package main

import (
	"fmt"
	"strings"
)

var availableWorkloads = []string{"tokenizer", "many-modules", "file-search"}

func workloadInput() []byte {
	return []byte(strings.Repeat("let count_01 = 123 + alpha;\nif (count_01 > 7) { beta_2 = count_01 - 1; }\n", 256))
}

func reference(input []byte, revision, modules int) string {
	tokens, checksum, previous := 0, 0, 0
	for _, b := range input {
		kind := 3
		switch {
		case b == 32 || b == 10 || b == 13 || b == 9:
			kind = 0
		case b >= 65 && b <= 90 || b >= 97 && b <= 122 || b == 95:
			kind = 1
		case b >= 48 && b <= 57:
			kind = 2
		}
		if kind != 0 && (kind != previous || kind == 3) {
			tokens++
		}
		previous = kind
		checksum = (checksum*31 + int(b)) % 1000003
	}
	checksum = (checksum + revision + 1) % 1000003
	for i := 1; i < modules; i++ {
		checksum = (checksum*(i+3) + i + 1) % 1000003
	}
	return fmt.Sprintf("tokens=%d checksum=%d\n", tokens, checksum)
}

func sources(language string, modules, revision int) map[string]string {
	files := map[string]string{}
	publicEdit := revision/5%2 == 1
	switch language {
	case "fairy":
		files["fairy.toml"] = "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"bench.footprint\"\nname = \"bench\"\nversion = \"0.1.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
		var imports, calls strings.Builder
		for i := 0; i < modules; i++ {
			name := fmt.Sprintf("m%03d", i)
			fmt.Fprintf(&imports, "import %s as %s;\n", name, name)
			argument, parameter := "", ""
			if i == 0 && publicEdit {
				argument = ", 0"
				parameter = ", i32 bias"
			}
			fmt.Fprintf(&calls, "    checksum = %s::mix(checksum%s);\n", name, argument)
			body := fmt.Sprintf("return (value * %d + %d) %% 1000003;", i+3, i+1)
			private := ""
			if i == 0 {
				private = fmt.Sprintf("fn i32 delta() { return %d; }\n", revision+1)
				body = "return (value + delta()) % 1000003;"
				if publicEdit {
					body = "return (value + delta() + bias) % 1000003;"
				}
			}
			files["src/"+name+".fy"] = fmt.Sprintf("module %s;\n%spub fn i32 mix(i32 value%s) { %s }\n", name, private, parameter, body)
		}
		files["src/main.fy"] = "module main;\nimport std::io as io;\n" + imports.String() + `@effects(console)
fn i32 main() {
    var u8[4096] buffer = [0; 4096];
    var i32 tokens = 0; var i32 checksum = 0; var i32 previous = 0;
    while (true) {
        let usize n = switch (io::read_stdin(ref mut buffer)) { case .ok(n) : { n } case .err(_) : { assert(false); 0 } };
        if (n == 0) { break; }
        var usize i = 0;
        while (i < n) {
            let i32 b = buffer[i] as i32; var i32 kind = 3;
            if (b == 32 || b == 10 || b == 13 || b == 9) { kind = 0; }
            else if ((b >= 65 && b <= 90) || (b >= 97 && b <= 122) || b == 95) { kind = 1; }
            else if (b >= 48 && b <= 57) { kind = 2; }
            if (kind != 0 && (kind != previous || kind == 3)) { tokens += 1; }
            previous = kind; checksum = (checksum * 31 + b) % 1000003; i += 1;
        }
    }
` + calls.String() + `    if (!io::write("tokens=").is_ok || !io::write_i64(tokens as i64).is_ok || !io::write(" checksum=").is_ok || !io::write_i64(checksum as i64).is_ok || !io::write("\n").is_ok) { return 1; }
    return 0;
}
`
	case "rust":
		files["Cargo.toml"] = "[package]\nname = \"bench\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
		var imports, calls strings.Builder
		for i := 0; i < modules; i++ {
			name := fmt.Sprintf("m%03d", i)
			fmt.Fprintf(&imports, "mod %s;\n", name)
			argument, parameter := "", ""
			if i == 0 && publicEdit {
				argument = ", 0"
				parameter = ", bias: i32"
			}
			fmt.Fprintf(&calls, "    checksum = %s::mix(checksum%s);\n", name, argument)
			body := fmt.Sprintf("(value * %d + %d) %% 1000003", i+3, i+1)
			private := ""
			if i == 0 {
				private = fmt.Sprintf("fn delta() -> i32 { %d }\n", revision+1)
				body = "(value + delta()) % 1000003"
				if publicEdit {
					body = "(value + delta() + bias) % 1000003"
				}
			}
			files["src/"+name+".rs"] = fmt.Sprintf("%spub fn mix(value: i32%s) -> i32 { %s }\n", private, parameter, body)
		}
		files["src/main.rs"] = imports.String() + `use std::io::{self, Read};
fn main() {
    let mut input = Vec::new(); io::stdin().read_to_end(&mut input).unwrap();
    let (mut tokens, mut checksum, mut previous): (i32, i32, i32) = (0, 0, 0);
    for byte in input {
        let b = byte as i32;
        let kind = if b == 32 || b == 10 || b == 13 || b == 9 { 0 }
            else if (65..=90).contains(&b) || (97..=122).contains(&b) || b == 95 { 1 }
            else if (48..=57).contains(&b) { 2 } else { 3 };
        if kind != 0 && (kind != previous || kind == 3) { tokens += 1; }
        previous = kind; checksum = (checksum * 31 + b) % 1000003;
    }
` + calls.String() + "    println!(\"tokens={} checksum={}\", tokens, checksum);\n}\n"
	case "zig":
		files["build.zig"] = `const std = @import("std");
pub fn build(b: *std.Build) void {
    const exe = b.addExecutable(.{ .name = "bench", .root_module = b.createModule(.{
        .root_source_file = b.path("src/main.zig"), .target = b.standardTargetOptions(.{}),
        .optimize = b.standardOptimizeOption(.{}), .link_libc = true,
    }) });
    b.installArtifact(exe);
}
`
		var imports, calls strings.Builder
		for i := 0; i < modules; i++ {
			name := fmt.Sprintf("m%03d", i)
			fmt.Fprintf(&imports, "const %s = @import(\"%s.zig\");\n", name, name)
			argument, parameter := "", ""
			if i == 0 && publicEdit {
				argument = ", 0"
				parameter = ", bias: i32"
			}
			fmt.Fprintf(&calls, "    checksum = %s.mix(checksum%s);\n", name, argument)
			body := fmt.Sprintf("return @mod(value * %d + %d, 1000003);", i+3, i+1)
			private := ""
			if i == 0 {
				private = fmt.Sprintf("fn delta() i32 { return %d; }\n", revision+1)
				body = "return @mod(value + delta(), 1000003);"
				if publicEdit {
					body = "return @mod(value + delta() + bias, 1000003);"
				}
			}
			files["src/"+name+".zig"] = fmt.Sprintf("%spub fn mix(value: i32%s) i32 { %s }\n", private, parameter, body)
		}
		files["src/main.zig"] = imports.String() + `extern "c" fn getchar() c_int;
extern "c" fn printf([*:0]const u8, ...) c_int;
pub fn main() void {
    var tokens: i32 = 0; var checksum: i32 = 0; var previous: i32 = 0;
    while (true) {
        const b: i32 = getchar(); if (b < 0) break;
        const kind: i32 = if (b == 32 or b == 10 or b == 13 or b == 9) 0
            else if ((b >= 65 and b <= 90) or (b >= 97 and b <= 122) or b == 95) 1
            else if (b >= 48 and b <= 57) 2 else 3;
        if (kind != 0 and (kind != previous or kind == 3)) tokens += 1;
        previous = kind; checksum = @mod(checksum * 31 + b, 1000003);
    }
` + calls.String() + "    _ = printf(\"tokens=%d checksum=%d\\n\", tokens, checksum);\n}\n"
	case "c":
		var headers, calls strings.Builder
		for i := 0; i < modules; i++ {
			name := fmt.Sprintf("m%03d", i)
			argument, parameter := "", ""
			if i == 0 && publicEdit {
				argument = ", 0"
				parameter = ", int bias"
			}
			fmt.Fprintf(&headers, "int %s_mix(int value%s);\n", name, parameter)
			fmt.Fprintf(&calls, "    checksum = %s_mix(checksum%s);\n", name, argument)
			body := fmt.Sprintf("return (value * %d + %d) %% 1000003;", i+3, i+1)
			private := ""
			if i == 0 {
				private = fmt.Sprintf("static int delta(void) { return %d; }\n", revision+1)
				body = "return (value + delta()) % 1000003;"
				if publicEdit {
					body = "return (value + delta() + bias) % 1000003;"
				}
			}
			files["src/"+name+".c"] = fmt.Sprintf("#include \"modules.h\"\n%sint %s_mix(int value%s) { %s }\n", private, name, parameter, body)
		}
		files["src/modules.h"] = headers.String()
		files["src/main.c"] = "#include <stdio.h>\n#include \"modules.h\"\n" + `int main(void) {
    int tokens = 0, checksum = 0, previous = 0, b;
    while ((b = getchar()) != EOF) {
        int kind = (b == 32 || b == 10 || b == 13 || b == 9) ? 0 :
            ((b >= 65 && b <= 90) || (b >= 97 && b <= 122) || b == 95) ? 1 :
            (b >= 48 && b <= 57) ? 2 : 3;
        if (kind != 0 && (kind != previous || kind == 3)) tokens++;
        previous = kind; checksum = (checksum * 31 + b) % 1000003;
    }
` + calls.String() + "    printf(\"tokens=%d checksum=%d\\n\", tokens, checksum); return 0;\n}\n"
	}
	return files
}
