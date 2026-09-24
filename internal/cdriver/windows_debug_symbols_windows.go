//go:build windows

package cdriver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"github.com/0disoft/fairyc/internal/pecoff"
)

const debugSymbolNameLimit = 1024

var debugSymbolLookupMu sync.Mutex

type WindowsDebugSymbol struct {
	RVA          uint32
	Name         string
	Displacement uint64
	Found        bool
}

type windowsDebugSymbolInfo struct {
	SizeOfStruct uint32
	TypeIndex    uint32
	Reserved     [2]uint64
	Index        uint32
	Size         uint32
	ModBase      uint64
	Flags        uint32
	Value        uint64
	Address      uint64
	Register     uint32
	Scope        uint32
	Tag          uint32
	NameLen      uint32
	MaxNameLen   uint32
	Name         [1]byte
}

// ResolveWindowsDebugSymbols looks up exact or nearest PDB symbols at selected
// function starts. Callers must validate release/PDB interval alignment first.
func ResolveWindowsDebugSymbols(artifact WindowsHostArtifact, rvas []uint32) (results []WindowsDebugSymbol, resultErr error) {
	if len(artifact.Bytes) == 0 || len(artifact.ProgramDatabase) == 0 || len(rvas) == 0 || len(rvas) > 4096 {
		return nil, fmt.Errorf("debug symbol lookup inputs are empty or too large")
	}
	if _, err := pecoff.InspectProgramDatabaseAssociation(artifact.Bytes, artifact.ProgramDatabase); err != nil {
		return nil, fmt.Errorf("debug symbol PDB identity: %w", err)
	}
	debugSymbolLookupMu.Lock()
	defer debugSymbolLookupMu.Unlock()
	systemDirectory, err := windowsSystemDirectoryForSymbols()
	if err != nil {
		return nil, err
	}
	dll := syscall.NewLazyDLL(filepath.Join(systemDirectory, "dbghelp.dll"))
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("load system DbgHelp: %w", err)
	}
	root, err := os.MkdirTemp("", "fairyc-debug-symbols-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := os.RemoveAll(root); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove debug symbol temp root: %w", err))
		}
	}()
	imagePath := filepath.Join(root, "program.exe")
	if err := os.WriteFile(imagePath, artifact.Bytes, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(root, "program.pdb"), artifact.ProgramDatabase, 0o600); err != nil {
		return nil, err
	}
	searchPath, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return nil, err
	}
	imagePathW, err := syscall.UTF16PtrFromString(imagePath)
	if err != nil {
		return nil, err
	}
	process := ^uintptr(0)
	initialized, _, initializeErr := dll.NewProc("SymInitializeW").Call(process, uintptr(unsafe.Pointer(searchPath)), 0)
	if initialized == 0 {
		return nil, fmt.Errorf("initialize DbgHelp: %w", initializeErr)
	}
	defer func() {
		if cleaned, _, cleanupErr := dll.NewProc("SymCleanup").Call(process); cleaned == 0 {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean up DbgHelp: %w", cleanupErr))
		}
	}()
	dll.NewProc("SymSetOptions").Call(0x00000002)
	moduleBase, _, loadErr := dll.NewProc("SymLoadModuleExW").Call(process, 0, uintptr(unsafe.Pointer(imagePathW)), 0, 0, 0, 0, 0)
	if moduleBase == 0 {
		return nil, fmt.Errorf("load debug symbol module: %w", loadErr)
	}
	lookup := dll.NewProc("SymFromAddr")
	results = make([]WindowsDebugSymbol, 0, len(rvas))
	for _, rva := range rvas {
		storage := make([]uint64, (int(unsafe.Sizeof(windowsDebugSymbolInfo{}))+debugSymbolNameLimit+7)/8)
		symbol := (*windowsDebugSymbolInfo)(unsafe.Pointer(&storage[0]))
		symbol.SizeOfStruct = uint32(unsafe.Sizeof(windowsDebugSymbolInfo{}))
		symbol.MaxNameLen = debugSymbolNameLimit
		var displacement uint64
		found, _, _ := lookup.Call(process, moduleBase+uintptr(rva), uintptr(unsafe.Pointer(&displacement)), uintptr(unsafe.Pointer(symbol)))
		row := WindowsDebugSymbol{RVA: rva}
		if found != 0 {
			if symbol.NameLen == 0 || symbol.NameLen >= debugSymbolNameLimit {
				return nil, fmt.Errorf("DbgHelp returned invalid symbol name length %d", symbol.NameLen)
			}
			row.Name = string(unsafe.Slice(&symbol.Name[0], int(symbol.NameLen)))
			row.Displacement = displacement
			row.Found = true
		}
		results = append(results, row)
	}
	return results, nil
}

func windowsSystemDirectoryForSymbols() (string, error) {
	getSystemDirectory := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemDirectoryW")
	buffer := make([]uint16, 32*1024)
	length, _, callErr := getSystemDirectory.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if length == 0 {
		return "", fmt.Errorf("query Windows system directory: %w", callErr)
	}
	if length >= uintptr(len(buffer)) {
		return "", fmt.Errorf("Windows system directory exceeds bound")
	}
	path := filepath.Clean(syscall.UTF16ToString(buffer[:length]))
	if path == "." || !filepath.IsAbs(path) {
		return "", fmt.Errorf("Windows system directory is invalid")
	}
	return path, nil
}
