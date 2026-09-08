package mircodec

import "github.com/0disoft/fairyc/internal/mir"

// StorageTransferSchema is a separate compiler-private envelope, not ordinary
// artifact, package-cache, source-interface or target compatibility admission.
const StorageTransferSchema = "fairyc.mir-storage-transfer-artifact/v0"

// EncodeStorageTransfers verifies explicit private storage operations before
// encoding. The caller owns the source/target context and positive byte limit.
func EncodeStorageTransfers(program mir.Program, maxBytes uint64) ([]byte, error) {
	return encode(program, maxBytes, StorageTransferSchema, mir.VerifyStorageTransfers)
}

// DecodeStorageTransfers accepts only the private transfer envelope and reruns
// its verifier before returning MIR. It does not publish generic provenance or
// establish that the artifact matches the caller's current package closure.
func DecodeStorageTransfers(encoded []byte, maxBytes uint64) (mir.Program, error) {
	return decode(encoded, maxBytes, StorageTransferSchema, mir.VerifyStorageTransfers)
}
