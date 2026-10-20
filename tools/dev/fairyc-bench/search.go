package main

import (
	"bytes"
	"fmt"
	"strings"
)

// Corpus files live beside the generated project, outside measured outputs.
func searchInput() []byte {
	prefix := strings.Repeat("x", 4094) + "error across a read boundary\n" + strings.Repeat("z", 9000) + "error\r\n\n"
	block := "info: ready\nerror: retry\r\nwarning: ERROR ignored\nerrorerror once\nbytes:\x00error utf8:한국어\n"
	tail := "error without a final newline"
	data := prefix + strings.Repeat(block, (1<<20)/len(block)+1)
	return []byte(data[:(1<<20)-len(tail)] + tail)
}

func searchReference(input []byte, revision int) string {
	lines, matches, checksum := 0, 0, 0
	for _, b := range input {
		checksum = (checksum*31 + int(b)) % 1000003
	}
	for _, line := range bytes.SplitAfter(input, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		lines++
		if bytes.Contains(line, []byte("error")) {
			matches++
		}
	}
	return fmt.Sprintf("lines=%d matches=%d checksum=%d\n", lines, matches, (checksum+revision+1)%1000003)
}

// Keep the existing revision helper and public-signature edits, replacing only
// the entry point with an actual buffered file-search CLI: bench FILE.
func searchSources(language string, revision int) map[string]string {
	files := sources(language, 1, revision)
	argument := ""
	if revision/5%2 == 1 {
		argument = ", 0"
	}
	switch language {
	case "fairy":
		files["src/main.fy"] = `module main;
import std::io as io;
import std::fs as fs;
import std::process as process;
import std::text as text;
import m000 as m000;
@effects(filesystem, console)
fn i32 search(fs::File input) {
    var fs::File file = input;
    var u8[4096] buffer = [0; 4096];
    let u8[5] needle = [101, 114, 114, 111, 114];
    var usize matched = 0;
    var bool found = false; var bool pending = false;
    var i32 lines = 0; var i32 matches = 0; var i32 checksum = 0;
    while (true) {
        let usize n = switch (fs::read(ref mut file, ref mut buffer)) {
            case .ok(n) : { n } case .err(_) : { return 1; }
        };
        if (n == 0) { break; }
        var usize i = 0;
        while (i < n) {
            let u8 b = buffer[i];
            checksum = (checksum * 31 + b as i32) % 1000003;
            if (b == 10) {
                lines += 1; if (found) { matches += 1; }
                matched = 0; found = false; pending = false;
            } else {
                pending = true;
                if (!found) {
                    if (b == needle[matched]) { matched += 1; }
                    else { matched = if (b == 101) { 1 } else { 0 }; }
                    if (matched == 5) { found = true; }
                }
            }
            i += 1;
        }
    }
    if (!fs::close(file).is_ok) { return 1; }
    if (pending) { lines += 1; if (found) { matches += 1; } }
    checksum = m000::mix(checksum` + argument + `);
    if (!io::write("lines=").is_ok || !io::write_i64(lines as i64).is_ok || !io::write(" matches=").is_ok || !io::write_i64(matches as i64).is_ok || !io::write(" checksum=").is_ok || !io::write_i64(checksum as i64).is_ok || !io::write("\n").is_ok) { return 1; }
    return 0;
}
@effects(filesystem, console)
fn i32 main() {
    if (process::arg_count() != 2) { return 2; }
    var u8[4096] path = [0; 4096];
    let usize n = switch (process::arg(1, ref mut path)) { case .ok(n) : { n } case .err(_) : { return 2; } };
    let ref u8[] path_view = ref path;
    return switch (text::validate_utf8(path_view[..n])) {
        case .ok(name) : { switch (fs::open_read_utf8(name)) { case .ok(file) : { search(file) } case .err(_) : { 1 } } }
        case .err(_) : { 2 }
    };
}
`
	case "rust":
		files["src/main.rs"] = `mod m000;
use std::{env, fs::File, io::{self, Read}, process};
fn search() -> io::Result<()> {
    let args: Vec<_> = env::args_os().collect();
    if args.len() != 2 { process::exit(2); }
    let mut file = File::open(&args[1])?;
    let mut buffer = [0u8; 4096];
    let needle = b"error";
    let (mut matched, mut found, mut pending) = (0usize, false, false);
    let (mut lines, mut matches, mut checksum) = (0i32, 0i32, 0i32);
    loop {
        let n = file.read(&mut buffer)?;
        if n == 0 { break; }
        for &b in &buffer[..n] {
            checksum = (checksum * 31 + b as i32) % 1000003;
            if b == 10 {
                lines += 1; if found { matches += 1; }
                matched = 0; found = false; pending = false;
            } else {
                pending = true;
                if !found {
                    if b == needle[matched] { matched += 1; }
                    else { matched = if b == b'e' { 1 } else { 0 }; }
                    if matched == 5 { found = true; }
                }
            }
        }
    }
    if pending { lines += 1; if found { matches += 1; } }
    checksum = m000::mix(checksum` + argument + `);
    println!("lines={} matches={} checksum={}", lines, matches, checksum);
    Ok(())
}
fn main() { if search().is_err() { process::exit(1); } }
`
	case "c":
		files["src/main.c"] = `#include <stdio.h>
#include "modules.h"
int main(int argc, char **argv) {
    if (argc != 2) return 2;
    FILE *file = fopen(argv[1], "rb"); if (!file) return 1;
    unsigned char buffer[4096]; const unsigned char needle[] = "error";
    int lines = 0, matches = 0, checksum = 0, matched = 0, found = 0, pending = 0;
    size_t n;
    while ((n = fread(buffer, 1, sizeof buffer, file)) != 0) {
        for (size_t i = 0; i < n; ++i) {
            unsigned char b = buffer[i]; checksum = (checksum * 31 + b) % 1000003;
            if (b == 10) {
                ++lines; if (found) ++matches;
                matched = 0; found = 0; pending = 0;
            } else {
                pending = 1;
                if (!found) {
                    if (b == needle[matched]) ++matched;
                    else matched = b == 'e' ? 1 : 0;
                    if (matched == 5) found = 1;
                }
            }
        }
    }
    int failed = ferror(file); if (fclose(file) != 0 || failed) return 1;
    if (pending) { ++lines; if (found) ++matches; }
    checksum = m000_mix(checksum` + argument + `);
    return printf("lines=%d matches=%d checksum=%d\n", lines, matches, checksum) < 0 ? 1 : 0;
}
`
	case "zig":
		files["src/main.zig"] = `const m000 = @import("m000.zig");
extern "c" fn fopen([*:0]const u8, [*:0]const u8) ?*anyopaque;
extern "c" fn fread([*]u8, usize, usize, *anyopaque) usize;
extern "c" fn ferror(*anyopaque) c_int;
extern "c" fn fclose(*anyopaque) c_int;
extern "c" fn printf([*:0]const u8, ...) c_int;
pub export fn main(argc: c_int, argv: [*][*:0]const u8) c_int {
    if (argc != 2) return 2;
    const file = fopen(argv[1], "rb") orelse return 1;
    var buffer: [4096]u8 = undefined; const needle = "error";
    var matched: usize = 0; var found = false; var pending = false;
    var lines: i32 = 0; var matches: i32 = 0; var checksum: i32 = 0;
    while (true) {
        const n = fread(&buffer, 1, buffer.len, file);
        if (n == 0) break;
        for (buffer[0..n]) |b| {
            checksum = @mod(checksum * 31 + @as(i32, b), 1000003);
            if (b == 10) {
                lines += 1; if (found) { matches += 1; }
                matched = 0; found = false; pending = false;
            } else {
                pending = true;
                if (!found) {
                    if (b == needle[matched]) { matched += 1; }
                    else { matched = if (b == 'e') 1 else 0; }
                    if (matched == 5) { found = true; }
                }
            }
        }
    }
    const failed = ferror(file); const closed = fclose(file);
    if (failed != 0 or closed != 0) return 1;
    if (pending) { lines += 1; if (found) { matches += 1; } }
    checksum = m000.mix(checksum` + argument + `);
    return if (printf("lines=%d matches=%d checksum=%d\n", lines, matches, checksum) < 0) 1 else 0;
}
`
	}
	return files
}
