//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

var getSystemDirectoryW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemDirectoryW")

const (
	symOptUndname   = 0x00000002
	symOptLoadLines = 0x00000010
	maxSymbolName   = 1024
)

type dbgHelpAPI struct {
	symInitializeW        *syscall.LazyProc
	symCleanup            *syscall.LazyProc
	symSetOptions         *syscall.LazyProc
	symLoadModuleExW      *syscall.LazyProc
	symFromAddr           *syscall.LazyProc
	symGetLineFromAddrW64 *syscall.LazyProc
}

type dbgHelpSymbolInfo struct {
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

type dbgHelpLineW64 struct {
	SizeOfStruct uint32
	Key          unsafe.Pointer
	LineNumber   uint32
	FileName     *uint16
	Address      uint64
}

func symbolicateWindowsProgram(image, pdb []byte, rva uint32) (receipt windowsSymbolicationReceipt, resultErr error) {
	if runtime.GOARCH != "amd64" {
		return windowsSymbolicationReceipt{}, fmt.Errorf("Windows symbolication requires windows/amd64, got windows/%s", runtime.GOARCH)
	}
	if len(image) == 0 || len(pdb) == 0 || rva == 0 {
		return windowsSymbolicationReceipt{}, fmt.Errorf("Windows symbolication inputs are empty")
	}
	api, err := loadSystemDbgHelp()
	if err != nil {
		return windowsSymbolicationReceipt{}, err
	}
	root, err := os.MkdirTemp("", "fairyc-symbol-canary-")
	if err != nil {
		return windowsSymbolicationReceipt{}, fmt.Errorf("create symbolication temp root: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(root); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove symbolication temp root: %w", err))
		}
	}()
	imagePath := filepath.Join(root, "program.exe")
	pdbPath := filepath.Join(root, "program.pdb")
	if err := os.WriteFile(imagePath, image, 0o600); err != nil {
		return windowsSymbolicationReceipt{}, fmt.Errorf("write symbolication image: %w", err)
	}
	if err := os.WriteFile(pdbPath, pdb, 0o600); err != nil {
		return windowsSymbolicationReceipt{}, fmt.Errorf("write symbolication PDB: %w", err)
	}
	searchPath, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return windowsSymbolicationReceipt{}, fmt.Errorf("encode symbol search path: %w", err)
	}
	imagePathW, err := syscall.UTF16PtrFromString(imagePath)
	if err != nil {
		return windowsSymbolicationReceipt{}, fmt.Errorf("encode symbol image path: %w", err)
	}
	process := syscall.Handle(^uintptr(0))
	initialized, _, initializeErr := api.symInitializeW.Call(uintptr(process), uintptr(unsafe.Pointer(searchPath)), 0)
	if initialized == 0 {
		return windowsSymbolicationReceipt{}, fmt.Errorf("initialize DbgHelp symbols: %w", initializeErr)
	}
	defer func() {
		cleaned, _, cleanupErr := api.symCleanup.Call(uintptr(process))
		if cleaned == 0 {
			resultErr = errors.Join(resultErr, windowsCallError("clean up DbgHelp symbols", cleanupErr))
		}
	}()
	api.symSetOptions.Call(symOptUndname | symOptLoadLines)
	moduleBase, _, loadErr := api.symLoadModuleExW.Call(
		uintptr(process), 0, uintptr(unsafe.Pointer(imagePathW)), 0, 0, 0, 0, 0,
	)
	if moduleBase == 0 {
		return windowsSymbolicationReceipt{}, fmt.Errorf("load PE/PDB module into DbgHelp: %w", loadErr)
	}
	address := uint64(moduleBase) + uint64(rva)
	storage := make([]uint64, (int(unsafe.Sizeof(dbgHelpSymbolInfo{}))+maxSymbolName+7)/8)
	symbol := (*dbgHelpSymbolInfo)(unsafe.Pointer(&storage[0]))
	symbol.SizeOfStruct = uint32(unsafe.Sizeof(dbgHelpSymbolInfo{}))
	symbol.MaxNameLen = maxSymbolName
	var displacement uint64
	found, _, symbolErr := api.symFromAddr.Call(
		uintptr(process), uintptr(address), uintptr(unsafe.Pointer(&displacement)), uintptr(unsafe.Pointer(symbol)),
	)
	if found == 0 {
		return windowsSymbolicationReceipt{}, fmt.Errorf("resolve DbgHelp symbol at RVA %#x: %w", rva, symbolErr)
	}
	if symbol.NameLen == 0 || symbol.NameLen >= maxSymbolName {
		return windowsSymbolicationReceipt{}, fmt.Errorf("DbgHelp returned invalid symbol name length %d", symbol.NameLen)
	}
	nameBytes := unsafe.Slice(&symbol.Name[0], int(symbol.NameLen))
	line := dbgHelpLineW64{SizeOfStruct: uint32(unsafe.Sizeof(dbgHelpLineW64{}))}
	var lineDisplacement uint32
	lineFound, _, lineErr := api.symGetLineFromAddrW64.Call(
		uintptr(process), uintptr(address), uintptr(unsafe.Pointer(&lineDisplacement)), uintptr(unsafe.Pointer(&line)),
	)
	if lineFound == 0 {
		return windowsSymbolicationReceipt{}, fmt.Errorf("resolve DbgHelp source line at RVA %#x: %w", rva, lineErr)
	}
	if line.FileName == nil || line.LineNumber == 0 || lineDisplacement != 0 {
		return windowsSymbolicationReceipt{}, fmt.Errorf("DbgHelp returned incomplete exact source-line evidence")
	}
	sourceFile, err := boundedUTF16PointerString(line.FileName, 32*1024)
	if err != nil {
		return windowsSymbolicationReceipt{}, err
	}
	return windowsSymbolicationReceipt{
		Name: string(nameBytes), Displacement: displacement,
		SourceFile: sourceFile, Line: line.LineNumber,
	}, nil
}

func loadSystemDbgHelp() (dbgHelpAPI, error) {
	systemDirectory, err := queryWindowsSystemDirectory()
	if err != nil {
		return dbgHelpAPI{}, err
	}
	path := filepath.Join(systemDirectory, "dbghelp.dll")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return dbgHelpAPI{}, fmt.Errorf("Windows system DbgHelp is unavailable")
	}
	dll := syscall.NewLazyDLL(path)
	if err := dll.Load(); err != nil {
		return dbgHelpAPI{}, fmt.Errorf("load Windows system DbgHelp: %w", err)
	}
	return dbgHelpAPI{
		symInitializeW: dll.NewProc("SymInitializeW"), symCleanup: dll.NewProc("SymCleanup"),
		symSetOptions: dll.NewProc("SymSetOptions"), symLoadModuleExW: dll.NewProc("SymLoadModuleExW"),
		symFromAddr: dll.NewProc("SymFromAddr"), symGetLineFromAddrW64: dll.NewProc("SymGetLineFromAddrW64"),
	}, nil
}

func queryWindowsSystemDirectory() (string, error) {
	buffer := make([]uint16, 32*1024)
	length, _, callErr := getSystemDirectoryW.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if length == 0 {
		return "", windowsCallError("query Windows system directory", callErr)
	}
	if length >= uintptr(len(buffer)) {
		return "", fmt.Errorf("Windows system directory exceeds %d UTF-16 units", len(buffer)-1)
	}
	path := filepath.Clean(syscall.UTF16ToString(buffer[:length]))
	if path == "." || !filepath.IsAbs(path) {
		return "", fmt.Errorf("Windows system directory is missing or not absolute")
	}
	return path, nil
}

func windowsCallError(action string, callErr error) error {
	if errno, ok := callErr.(syscall.Errno); ok && errno == 0 {
		return fmt.Errorf("%s failed without a Win32 error", action)
	}
	return fmt.Errorf("%s: %w", action, callErr)
}

func boundedUTF16PointerString(value *uint16, limit int) (string, error) {
	if value == nil || limit <= 0 {
		return "", fmt.Errorf("DbgHelp returned an invalid source path pointer")
	}
	units := make([]uint16, 0, 260)
	for index := 0; index < limit; index++ {
		unit := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(value)) + uintptr(index)*2))
		if unit == 0 {
			if len(units) == 0 {
				return "", fmt.Errorf("DbgHelp returned an empty source path")
			}
			return syscall.UTF16ToString(units), nil
		}
		units = append(units, unit)
	}
	return "", fmt.Errorf("DbgHelp source path exceeds %d UTF-16 units", limit)
}
