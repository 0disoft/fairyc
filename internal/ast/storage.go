package ast

import "github.com/0disoft/fairyc/internal/syntax"

// PrivateStorageLayout is produced only by private generic expansion, never by
// the parser. Type is the unevaluated query operand; Text selects size/alignment.
const PrivateStorageLayout syntax.Kind = "private-storage-layout"

// PrivateStorageTake carries one unsafe raw-storage ownership transfer.
const PrivateStorageTake syntax.Kind = "private-storage-take"

// PrivateStoragePut consumes one value into caller-owned raw storage.
const PrivateStoragePut syntax.Kind = "private-storage-put"

// PrivateStorageBorrow ties a raw element address to the owner's shared or
// exclusive borrow lifetime. Text selects the private borrow mode.
const PrivateStorageBorrow syntax.Kind = "private-storage-borrow"

// PrivateStorageElementAddress is an SDK-only unsafe address calculation for
// one initialized element; its caller proves the allocation extent.
const PrivateStorageElementAddress syntax.Kind = "private-storage-element-address"
