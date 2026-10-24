package main

import "fmt"

type hostSmokeReceipt struct {
	artifactBytes                  int
	validatedSections              int
	runtimeFunctions               int
	version2Infos                  int
	targetBuildExecutions          int
	workspaceBuildExecutions       int
	workspaceRunExecutions         int
	windowsFileSourceExecutions    int
	ownedTextSourceExecutions      int
	targetRunExecutions            int
	targetExecExecutions           int
	windowsFileExecutions          int
	bufferProbeExecutions          int
	ownedTextProbeExecutions       int
	arenaProbeExecutions           int
	regionFailureProbeImages       int
	regionFailureCases             int
	bufferFailureProbeImages       int
	bufferFailureCases             int
	readAppendFailureProbeImages   int
	readAppendFailureCases         int
	readToEndFailureProbeImages    int
	readToEndFailureCases          int
	stringAppendFailureProbeImages int
	stringAppendFailureCases       int
	cstringFailureProbeImages      int
	cstringFailureCases            int
	stringCopyFailureProbeImages   int
	stringCopyFailureCases         int
	stringFromBufferProbeImages    int
	stringFromBufferProbeCases     int
	stdFSBuildExecutions           int
	processArgsExecutions          int
	processArgsDecodeExecutions    int
	processArgsBuildImages         int
	processArgsDecodeImages        int
	nativeTestArgsExecutions       int
	nativeTestStdinExecutions      int
	nativeTestOutputExecutions     int
	nativeTestTimeoutExecutions    int
	stdIoScalarBuildExecutions     int
	cIntCastExecutions             int
	atlasCallProcessIterations     int
	atlasCallScaleShapes           int
	atlasCallRetainedBatches       int
	layoutCases                    int
	layoutExecutions               int
	abiCases                       int
	abiExecutions                  int
	relocationExecutions           int
	weakSearchExecutions           int
	archiveExecutions              int
	structResultExecutions         int
	unwindCases                    int
	unwindExecutions               int
	mixedStackExecutions           int
	version2EpilogExecutions       int
	standaloneCompilerBytes        int64
	toolchainVersion               string
}

func reportHostCorrectnessSuccess(receipt hostSmokeReceipt) {
	fmt.Printf(
		"Windows host correctness smoke passed: %d target builds, %d workspace builds, %d workspace runs, %d direct target runs, %d native File probes, and %d ABI/layout cases with msvc %s\n",
		receipt.targetBuildExecutions,
		receipt.workspaceBuildExecutions,
		receipt.workspaceRunExecutions,
		receipt.targetRunExecutions,
		receipt.windowsFileExecutions,
		receipt.abiCases+receipt.layoutCases,
		receipt.toolchainVersion,
	)
	fmt.Println("Windows host measurements skipped; use --with-measurements for the scheduled/manual measurement lane")
}

func reportHostSmokeSuccess(receipt hostSmokeReceipt) {
	// The epilog probe contributes two additional structurally inspected images.
	inspectedPEImages := 1 + receipt.targetBuildExecutions + receipt.workspaceBuildExecutions + receipt.windowsFileSourceExecutions + receipt.ownedTextSourceExecutions + receipt.targetRunExecutions + receipt.workspaceRunExecutions + receipt.targetExecExecutions + receipt.windowsFileExecutions + receipt.bufferProbeExecutions + receipt.ownedTextProbeExecutions + receipt.arenaProbeExecutions + receipt.regionFailureProbeImages + receipt.bufferFailureProbeImages + receipt.readAppendFailureProbeImages + receipt.readToEndFailureProbeImages + receipt.stringAppendFailureProbeImages + receipt.cstringFailureProbeImages + receipt.stringCopyFailureProbeImages + receipt.stringFromBufferProbeImages + receipt.processArgsBuildImages + receipt.processArgsDecodeImages + receipt.nativeTestArgsExecutions + receipt.nativeTestStdinExecutions + receipt.nativeTestOutputExecutions + receipt.nativeTestTimeoutExecutions + receipt.layoutExecutions + receipt.abiExecutions + receipt.relocationExecutions + receipt.weakSearchExecutions + receipt.archiveExecutions + receipt.structResultExecutions + receipt.cIntCastExecutions + receipt.unwindExecutions + receipt.mixedStackExecutions + receipt.version2EpilogExecutions
	fmt.Printf("Generated-Fairy Region failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering payload-allocation failure, node-allocation rollback, exact status 1, lexical reset, automatic Arena drop, and zero retained probe allocations\n", receipt.regionFailureCases, receipt.regionFailureProbeImages)
	fmt.Printf("Generated-Fairy Buffer failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering with_capacity initial-allocation failure plus resize_zeroed and append_byte growth failures with preserved len/capacity, exact status 1, and zero retained probe allocations\n", receipt.bufferFailureCases, receipt.bufferFailureProbeImages)
	fmt.Printf("Generated-Fairy read_append failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering growth-allocation failure before I/O, exact file/allocation statuses 0/1, preserved Buffer len/capacity, consuming File close, automatic Buffer drop, and zero retained probe allocations\n", receipt.readAppendFailureCases, receipt.readAppendFailureProbeImages)
	fmt.Printf("Generated-Fairy read_to_end failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering one committed 65536-byte prefix, second-window allocation failure before I/O, exact file/allocation statuses 0/1, preserved cursor and Buffer bytes, consuming File close, automatic Buffer drop, and zero retained probe allocations\n", receipt.readToEndFailureCases, receipt.readToEndFailureProbeImages)
	fmt.Printf("Generated-Fairy String append failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering successful copy allocation, append growth-allocation failure with preserved owner and len/is_empty/view, exact statuses 1/1, automatic String drop, and zero retained probe allocations\n", receipt.stringAppendFailureCases, receipt.stringAppendFailureProbeImages)
	fmt.Printf("Generated-Fairy CString failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering cstring_from_text initial-allocation failure, exact statuses 1/1, automatic CString drop, and zero retained probe allocations\n", receipt.cstringFailureCases, receipt.cstringFailureProbeImages)
	fmt.Printf("Generated-Fairy String copy failure proof passed: %d forced .err executions across %d inspected low/optimized MSVC images, covering text::copy initial-allocation failure, exact statuses 1/1, automatic String drop, and zero retained probe allocations\n", receipt.stringCopyFailureCases, receipt.stringCopyFailureProbeImages)
	fmt.Printf("Generated-Fairy Buffer/String zero-copy transfer proof passed: %d native executions across %d inspected low/optimized MSVC images, covering Buffer-to-String-to-mutable-Buffer allocation identity under a fail-after-two schedule, invalid UTF-8 ownership return with preserved len/capacity/bytes, empty round-trip success, automatic String and Buffer drop, and zero retained probe allocations\n", receipt.stringFromBufferProbeCases, receipt.stringFromBufferProbeImages)
	fmt.Printf("Generated-Fairy std::fs one-call write, owned-Buffer read append/read-to-end, file and directory creation/removal/no-replace rename, recursive parent creation, no-replace byte copy and hard-link creation, path-kind snapshot, absolute/current/end seek, same-handle read-only observation/mutation, hard-link, allocated-size, and compound FileMetadata observation, append, file-kind, exact-read, and file-identity proof passed: %d native executions across %d inspected low/optimized MSVC images, covering one-call prefix success, read-only write failure with zero progress, write-all completion, initialized-prefix Buffer append, current-position read-to-end, short read, EOF, write-only read failure with preserved bytes, open-or-create preservation, exclusive create collision, exact file remove success plus missing-path and open-sharing failures, no-replace file rename collision preservation plus open-sharing failure and exact final bytes, exact byte-copy completion plus collision preservation and missing-source no-create behavior, no-replace hard-link collision plus missing-source no-create behavior, shared identity, link-count and bytes, source-name removal, and surviving destination alias, path-level ordinary file and directory classification plus exact missing failure, one-level and nested directory creation, existing-directory success, missing-parent and file/directory collision failures, partial-prefix preservation after a later invalid-name failure, non-empty removal failure, child cleanup, exact empty-directory removal, no-replace directory rename collision, moved-child preservation, old-path disappearance, bounded negative seek, start-preserving failure, beyond-EOF cursor movement with unchanged size, write-capable read-only true/false mutation, exact read-only-handle access denial with unchanged metadata, ordinary and host-created read-only metadata plus both names of a host-created two-link file with sequential FileMetadata fields, filesystem-reported allocation at least logical size, and unchanged cursor/size, write-only append creation, seek-resistant EOF writes, exact retained bytes, disk files, the NUL character device, complete reads, unexpected-EOF prefix preservation, same-file handles, and distinct live files\n", receipt.stdFSBuildExecutions, receipt.stdFSBuildExecutions)
	fmt.Printf("std::fs recursive removal proof covered generated-Fairy exact four-object success, missing-path storage, open-child sharing failure with zero progress, read-only preservation, raw actual junction-leaf target survival, and retry cleanup across the same %d native executions\n", receipt.stdFSBuildExecutions)
	fmt.Printf("Stable run program-argument forwarding proof passed: %d native executions across %d inspected low/optimized MSVC images, covering argv[0] inclusion and ordering, empty, flag-looking, space/quote/trailing-backslash, and non-ASCII UTF-8 argument round-trips, out-of-range status 1, too-small status 3 with exact needed and an unchanged caller buffer, CLI embedded-NUL and delimiter rejection, and low-level invalid-surrogate status 2 with an unchanged buffer\n", receipt.processArgsExecutions+receipt.processArgsDecodeExecutions, receipt.processArgsBuildImages+receipt.processArgsDecodeImages)
	fmt.Printf("Stable native-test program-argument forwarding proof passed: %d isolated generated-Fairy executions across low/optimized MSVC, covering flag-looking, empty, and non-ASCII UTF-8 arguments after the first -- delimiter\n", receipt.nativeTestArgsExecutions)
	fmt.Printf("Stable native-test stdin proof passed: %d isolated generated-Fairy executions across low/optimized MSVC, covering direct single-test binary input with an embedded NUL plus bounded contained-file replay from byte zero to every selected test\n", receipt.nativeTestStdinExecutions)
	fmt.Printf("Stable native-test output attachment proof passed: %d isolated generated-Fairy executions across low/optimized MSVC, covering exact binary stdout/stderr base64, pass/trap outcomes, off-stream trap control, and continuation\n", receipt.nativeTestOutputExecutions)
	fmt.Printf("Stable native-test timeout proof passed: %d isolated generated-Fairy executions across low/optimized MSVC, covering explicit 1-second process termination, code 6, and no partial report\n", receipt.nativeTestTimeoutExecutions)
	fmt.Printf("std::io scalar output proof passed: %d native executions across %d inspected low/optimized MSVC images, covering stdout u32 0/1/max, usize 0/1/max, bool true/false, plus stderr i32 0/negative/min/max, u32 0/max, usize 0/max, and bool true/false exact ASCII bytes with exact written counts\n", receipt.stdIoScalarBuildExecutions, receipt.stdIoScalarBuildExecutions)
	fmt.Println("Generated-Fairy user-enum Result proof passed: 4 native executions across low/optimized MSVC, covering .ok enum construction, enum payload extraction, exhaustive enum switching, .err inert-struct extraction, and ? propagation of enum errors")
	fmt.Println("Generated-Fairy user-enum Option proof passed: 2 native executions across low/optimized MSVC, covering Some enum construction, payload extraction, exhaustive enum switching, and None")
	fmt.Println("Generated-Fairy copy fixed-array Option/Result proof passed: 2 native executions across low/optimized MSVC, covering Some and None, Ok and Err, indexed payload reads, and success/error propagation")
	fmt.Println("Generated-Fairy nested inert-enum proof passed: 2 native executions across low/optimized MSVC, covering nested construction, by-value payload transport, extraction, and exhaustive nested switching")
	fmt.Printf("Generated-Fairy c::int cast proof passed: %d native executions across low/optimized MSVC, covering imported negative-capable C int identity, explicit c::int-to-i32 conversion, exact value round-trip, explicit i32-to-c::int conversion, and a separate C consumer\n", receipt.cIntCastExecutions)
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes with %d validated sections, %d runtime-function records, and %d version-2 unwind records, %d source-level Windows target builds, %d package-workspace builds, %d package-workspace runs, %d source-level safe File lifecycle builds, %d source-level safe owned-text lifecycle builds, %d direct run-target executions, and %d exec-target executions checked LLP64 c.long emission, move-only pointer-width resource transport, closed SystemHeap, public safe std::bytes Buffer ownership and views, Arena/Region, String/CString/CStr, allocation-free owner-bound text-to-byte views, UTF-8 crossings, safe borrowed-view internal calls, byte lengths, typed scalar comparisons, scalar fixed-array calls, checked reads and mutable indexed stores, checked byte indexing and explicit or omitted-bound slicing, always-checked assertions, bounded lexical defer, value-preserving integer casts, checked unsigned arithmetic, division, remainder, bitwise operations, and left/right shifts, exact stdout bytes, stable-run stdin forwarding, bounded allocation-free stdin byte reads with EOF and partial-buffer evidence, and native result 42, including one %d-byte standalone Go-built fairyc.exe quickstart, output, locked workspace execution, semantic text/JSON tests, ten isolated native-test commands covering fourteen executions across low/optimized MSVC including exact portable test-file and recursive test-directory selection, exact-current source anchors, and optimized JSON trap continuation, selected workspace-member tests, and package-qualified workspace aggregation, two target-explicit C emissions, nine paired cold/warm persistent-cache process measurements, one nine-process bounded persistent-cache growth staircase, an eight-request editor-shaped workspace session with Windows process-memory peaks, %d fresh locked Atlas workspace-call catalog processes with per-process memory peaks, %d generated Atlas workspace-call scaling shapes, and %d post-GC retained-index batches, 44 run-host outcomes covered fixed arrays, nested inert enums, byte-view iteration and slicing, checked indexed stores, assertions and lexical defer in low and optimized profiles, usize arithmetic and overflow, unsigned division and zero traps, typed bitwise operations, checked shift-width and left-shift overflow traps, signed results, and six trap categories, Windows File partial I/O and consuming-close passed %d separate-translation-unit MSVC executions, Buffer initialization, reserve, self-overlap append, and zeroed-resize allocation rollback passed %d separate-translation-unit MSVC executions, owned-text opaque-storage failure preservation, forced allocation rollback, and repeated drop passed %d separate-translation-unit MSVC executions, Arena zeroing, two-stage allocation rollback, stale Region rejection, and active-drop preservation passed %d separate-translation-unit MSVC executions, %d LLP64 layout cases passed %d MSVC probe executions, %d Microsoft x64 ABI cases passed %d separate-translation-unit MSVC probe executions, exact object addends and final-image REL32 through REL32_5, weak-alias fallback/strong override, ADDR64, and unwind ADDR32NB values plus CodeView SECTION/SECREL pairs, PE/PDB identity, and exact DbgHelp function/line symbolication passed %d separate-translation-unit MSVC executions, weak NOLIBRARY suppression and LIBRARY extraction passed %d separate-translation-unit MSVC executions, archive extraction, pick-any ownership, selected-child retention, and discarded-child elimination passed %d MSVC input-order executions, generated-Fairy small and large struct results passed %d separate-translation-unit MSVC executions across C11 and C17 consumer modes, %d Windows x64 unwind cases plus exact profile-specific handler-tail ADDR32NB binding passed %d actual exception-dispatch and virtual-unwind executions, %d PE32+ target or probe images passed structural and unwind inspection, and every MSVC compilation object passed COFF inspection across low/optimized with msvc %s\n", receipt.artifactBytes, receipt.validatedSections, receipt.runtimeFunctions, receipt.version2Infos, receipt.targetBuildExecutions, receipt.workspaceBuildExecutions, receipt.workspaceRunExecutions, receipt.windowsFileSourceExecutions, receipt.ownedTextSourceExecutions, receipt.targetRunExecutions, receipt.targetExecExecutions, receipt.standaloneCompilerBytes, receipt.atlasCallProcessIterations, receipt.atlasCallScaleShapes, receipt.atlasCallRetainedBatches, receipt.windowsFileExecutions, receipt.bufferProbeExecutions, receipt.ownedTextProbeExecutions, receipt.arenaProbeExecutions, receipt.layoutCases, receipt.layoutExecutions, receipt.abiCases, receipt.abiExecutions, receipt.relocationExecutions, receipt.weakSearchExecutions, receipt.archiveExecutions, receipt.structResultExecutions, receipt.unwindCases, receipt.unwindExecutions, inspectedPEImages, receipt.toolchainVersion)
}
