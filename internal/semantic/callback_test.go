package semantic_test

import (
	"strings"
	"testing"
)

const callbackSource = `module main;
unsafe extern C fn i32 visit(i32 value, unsafe extern C fn i32(i32, ptr mut void) callback, ptr mut void data);
unsafe fn i32 handle(i32 value, ptr mut void data){return value+1;}
fn i32 main(){return unsafe{visit(41,handle,null)};}`

func TestBoundedCallbackTyping(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": callbackSource})
	assertCleanTypeResult(t, result)
}

func TestBoundedCallbackBoundaries(t *testing.T) {
	for name, source := range map[string]string{
		"safe-target":     strings.Replace(callbackSource, "unsafe fn i32 handle", "fn i32 handle", 1),
		"null-target":     strings.Replace(callbackSource, "41,handle,null", "41,null,null", 1),
		"wrong-result":    strings.Replace(callbackSource, "unsafe fn i32 handle(i32 value, ptr mut void data){return value+1;}", "unsafe fn u32 handle(i32 value, ptr mut void data){return 1;}", 1),
		"wrong-context":   strings.Replace(callbackSource, "ptr mut void data){return value+1;}", "ptr void data){return value+1;}", 1),
		"missing-data":    `module main;unsafe extern C fn i32 bad(unsafe extern C fn i32(i32, ptr mut void) callback);`,
		"wrong-position":  `module main;unsafe extern C fn i32 bad(unsafe extern C fn i32(i32, ptr mut void) callback,ptr mut void data,i32 extra);`,
		"local-storage":   `module main; fn i32 main(){let unsafe extern C fn i32(i32,ptr mut void) cb=null;return 0;}`,
		"callback-result": `module main;unsafe extern C fn unsafe extern C fn i32(i32,ptr mut void) bad();`,
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": source})
			if len(result.Diagnostics)+len(result.Unsupported) == 0 {
				t.Fatal("unbounded callback form admitted")
			}
		})
	}
}
