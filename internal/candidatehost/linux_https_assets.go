package candidatehost

import _ "embed"

// These worker assets are not selected by LinuxSource until the owning
// parent-side request/cancellation adapter is implemented and verified.
//
//go:embed linux_https_protocol.c
var linuxHTTPSProtocol string

//go:embed linux_https_worker.c
var linuxHTTPSWorker string

//go:embed linux_https_reader.c
var linuxHTTPSReader string
