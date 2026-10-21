package cbackendnative

// Compiler records the bounded toolchain fingerprint used by one native run.
type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Report is the deterministic success record emitted by the native fixture command.
type Report struct {
	SchemaVersion                       string     `json:"schemaVersion"`
	Target                              string     `json:"target"`
	Libc                                string     `json:"libc"`
	Compilers                           []Compiler `json:"compilers"`
	Profiles                            []string   `json:"profiles"`
	Cases                               []string   `json:"cases"`
	Executions                          int        `json:"executions"`
	DropTraceExecutions                 int        `json:"dropTraceExecutions"`
	LayoutCases                         []string   `json:"layoutCases"`
	LayoutProbeExecutions               int        `json:"layoutProbeExecutions"`
	Sanitizer                           string     `json:"sanitizer"`
	SanitizerExecutions                 int        `json:"sanitizerExecutions"`
	ABICases                            []string   `json:"abiCases"`
	ABIDirections                       []string   `json:"abiDirections"`
	ABIProbeExecutions                  int        `json:"abiProbeExecutions"`
	SystemHeapExecutions                int        `json:"systemHeapExecutions"`
	SystemHeapSourceCallExecutions      int        `json:"systemHeapSourceCallExecutions"`
	BufferExecutions                    int        `json:"bufferExecutions"`
	BufferSourceCallExecutions          int        `json:"bufferSourceCallExecutions"`
	ArenaExecutions                     int        `json:"arenaExecutions"`
	ArenaSourceCallExecutions           int        `json:"arenaSourceCallExecutions"`
	TextViewExecutions                  int        `json:"textViewExecutions"`
	OwnedTextExecutions                 int        `json:"ownedTextExecutions"`
	OwnedTextSourceCallExecutions       int        `json:"ownedTextSourceCallExecutions"`
	LinuxConsoleExecutions              int        `json:"linuxConsoleExecutions"`
	LinuxConsoleFailureExecutions       int        `json:"linuxConsoleFailureExecutions"`
	LinuxFileExecutions                 int        `json:"linuxFileExecutions"`
	LinuxFileSourceCallExecutions       int        `json:"linuxFileSourceCallExecutions"`
	LinuxPublicFileExecutions           int        `json:"linuxPublicFileExecutions"`
	ForeignCallExecutions               int        `json:"foreignCallExecutions"`
	ForeignCIntCallExecutions           int        `json:"foreignCIntCallExecutions"`
	ForeignCLongCallExecutions          int        `json:"foreignCLongCallExecutions"`
	ForeignCScalarFamilyCallExecutions  int        `json:"foreignCScalarFamilyCallExecutions"`
	ForeignCSizeTCallExecutions         int        `json:"foreignCSizeTCallExecutions"`
	ForeignCSSizeTCallExecutions        int        `json:"foreignCSSizeTCallExecutions"`
	ForeignPointerCallExecutions        int        `json:"foreignPointerCallExecutions"`
	ForeignCStructResultCallExecutions  int        `json:"foreignCStructResultCallExecutions"`
	TextValidationSourceCallExecutions  int        `json:"textValidationSourceCallExecutions"`
	TextViewOutputSourceCallExecutions  int        `json:"textViewOutputSourceCallExecutions"`
	CStrValidationSourceCallExecutions  int        `json:"cStrValidationSourceCallExecutions"`
	ByteStringSourceCallExecutions      int        `json:"byteStringLiteralSourceCallExecutions"`
	UTF8StringSourceCallExecutions      int        `json:"utf8StringLiteralSourceCallExecutions"`
	CStringLiteralSourceCallExecutions  int        `json:"cStringLiteralSourceCallExecutions"`
	BuildSmokeExecutions                int        `json:"buildSmokeExecutions"`
	StandardLibraryBuildSmokeExecutions int        `json:"standardLibraryBuildSmokeExecutions"`
	ByteStringBuildSmokeExecutions      int        `json:"byteStringLiteralBuildSmokeExecutions"`
	UTF8StringBuildSmokeExecutions      int        `json:"utf8StringLiteralBuildSmokeExecutions"`
	CStringLiteralBuildSmokeExecutions  int        `json:"cStringLiteralBuildSmokeExecutions"`
}
