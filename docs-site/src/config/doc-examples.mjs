/**
 * Canonical source-of-truth snippets mirrored across all locale docs and compiled by
 * scripts/check-doc-examples.mjs. Comment lines are normalized away when matching locales.
 */
export const docExamples = [
	{
		id: 'first-program-manifest',
		page: 'start/first-program.md',
		kind: 'toml',
		source: `schema = "fairyc.manifest/v1"

[package]
uid = "example.hello-fairy"
name = "hello-fairy"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"`,
	},
	{
		id: 'first-program-main',
		page: 'start/first-program.md',
		kind: 'fy',
		source: `module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}`,
	},
	{
		id: 'first-program-test',
		page: 'start/first-program.md',
		kind: 'fy',
		source: `module tests::main;

import main as app;

@test
fn void answer_is_42() {
    assert(app::answer() == 42);
}`,
	},
	{
		id: 'control-flow-closure',
		page: 'language/control-flow-and-functions.md',
		kind: 'fy',
		source: `fn i32 apply_once(fn i32(i32) f, i32 x) {
    return f(x);
}

fn i32 make_result() {
    let i32 offset = 7;
    let fn i32(i32) add_offset = fn [i32 offset] i32(i32 x) {
        return x + offset;
    };
    return apply_once(add_offset, 70);
}`,
	},
	{
		id: 'structs-parse-error-enum',
		page: 'language/structs-enums-and-errors.md',
		kind: 'fy',
		source: `enum ParseError {
    Empty;
    InvalidByte { u8 offset; u8 byte; };
}`,
	},
	{
		id: 'structs-parse-error-switch',
		page: 'language/structs-enums-and-errors.md',
		kind: 'fy',
		source: `fn i32 code_of(ParseError err) {
    return switch (err) {
        case .Empty: { 1 }
        case .InvalidByte { .offset = pos, .byte = _ }: {
            (pos as i32) + 10
        }
    };
}`,
	},
	{
		id: 'structs-add-one-result',
		page: 'language/structs-enums-and-errors.md',
		kind: 'fy',
		source: `fn Result<i32, ParseError> add_one(Result<i32, ParseError> input) {
    let i32 n = input?;
    return .ok(n + 1);
}`,
	},
	{
		id: 'memory-region-alloc',
		page: 'language/memory-and-resources.md',
		kind: 'fy',
		source: `import std::alloc as alloc;

fn i32 use_scratch(alloc::Arena arena) {
    var alloc::Arena mutable_arena = arena;
    region scratch in ref mut mutable_arena {
    }
    return 0;
}`,
	},
	{
		id: 'modules-app-math',
		page: 'language/modules-and-projects.md',
		kind: 'fy',
		source: `module app::math;

pub fn i32 add(i32 a, i32 b) {
    return a + b;
}`,
	},
	{
		id: 'modules-main',
		page: 'language/modules-and-projects.md',
		kind: 'fy',
		source: `module main;

import app::math as math;
import std::io as io;

@effects(console)
fn i32 main() {
    let i32 v = math::add(70, 7);
    discard io::write_i32(v);
    return 0;
}`,
	},
	{
		id: 'modules-manifest',
		page: 'language/modules-and-projects.md',
		kind: 'toml',
		source: `schema = "fairyc.manifest/v1"

[package]
uid = "example.my-app"
name = "my-app"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"

[workspace]
members = ["libs/answer"]

[dependencies.answer]
path = "libs/answer"
version = "0.1.0"
features = []`,
	},
	{
		id: 'win32-build-json',
		page: 'examples/c-interop-and-win32.md',
		kind: 'json',
		source: `{
  "schema": "fairyc.windows-build/v1",
  "subsystem": "windows",
  "libraries": ["user32.lib", "gdi32.lib", "imm32.lib"],
  "sources": ["native/window.c", "native/ime.c"],
  "resources": ["native/app.rc"]
}`,
	},
	{
		id: 'win32-app-toml',
		page: 'examples/c-interop-and-win32.md',
		kind: 'toml',
		source: `[app]
product_name = "My Tool"
company_name = "Example Co"
description = "My Tool desktop utility"
executable_name = "my-tool.exe"
package_name = "my-tool"
assembly_name = "example.my-tool"
icon = "native/app.ico"`,
	},
];
