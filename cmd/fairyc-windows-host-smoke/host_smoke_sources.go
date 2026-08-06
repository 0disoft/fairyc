package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func loadHostSmokeSources() (map[string]string, error) {
	successSource, err := os.ReadFile(filepath.FromSlash(checkedInSuccessPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows smoke source: %w", err)
	}
	moduleMainSource, err := os.ReadFile(filepath.FromSlash(checkedInModuleMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in multi-source entry: %w", err)
	}
	moduleSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInModuleSupportPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in multi-source support: %w", err)
	}
	windowsTargetSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsTargetPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows target source: %w", err)
	}
	windowsQuickstartSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsQuickstartPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows quickstart source: %w", err)
	}
	windowsOutputSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsOutputPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows output source: %w", err)
	}
	windowsOutputSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsOutputSupportPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows output support source: %w", err)
	}
	stdIOManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdIOManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::io workspace manifest: %w", err)
	}
	stdIOMain, err := os.ReadFile(filepath.FromSlash(checkedInStdIOMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::io workspace entry: %w", err)
	}
	stdIoScalarManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdIoScalarManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::io scalar workspace manifest: %w", err)
	}
	stdIoScalarMain, err := os.ReadFile(filepath.FromSlash(checkedInStdIoScalarMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::io scalar workspace entry: %w", err)
	}
	stdAllocManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdAllocManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::alloc workspace manifest: %w", err)
	}
	stdAllocMain, err := os.ReadFile(filepath.FromSlash(checkedInStdAllocMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::alloc workspace entry: %w", err)
	}
	stdBytesManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdBytesManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::bytes workspace manifest: %w", err)
	}
	stdBytesMain, err := os.ReadFile(filepath.FromSlash(checkedInStdBytesMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::bytes workspace entry: %w", err)
	}
	stdTextManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdTextManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::text workspace manifest: %w", err)
	}
	stdTextMain, err := os.ReadFile(filepath.FromSlash(checkedInStdTextMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::text workspace entry: %w", err)
	}
	stdTextFromBufferManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdTextFromBufferManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::text from-buffer workspace manifest: %w", err)
	}
	stdTextFromBufferMain, err := os.ReadFile(filepath.FromSlash(checkedInStdTextFromBufferMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::text from-buffer workspace entry: %w", err)
	}
	stdFSManifest, err := os.ReadFile(filepath.FromSlash(checkedInStdFSManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::fs workspace manifest: %w", err)
	}
	stdFSMain, err := os.ReadFile(filepath.FromSlash(checkedInStdFSMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::fs workspace entry: %w", err)
	}
	stdFSPathOps, err := os.ReadFile(filepath.FromSlash(checkedInStdFSPathOpsPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::fs path operations: %w", err)
	}
	stdFSPositioning, err := os.ReadFile(filepath.FromSlash(checkedInStdFSPositioningPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::fs positioning operations: %w", err)
	}
	stdFSMetadata, err := os.ReadFile(filepath.FromSlash(checkedInStdFSMetadataPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::fs metadata operations: %w", err)
	}
	stdFSLinkOps, err := os.ReadFile(filepath.FromSlash(checkedInStdFSLinkOpsPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in std::fs hard-link operations: %w", err)
	}
	regionsManifest, err := os.ReadFile(filepath.FromSlash(checkedInRegionsManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Region workspace manifest: %w", err)
	}
	regionsMain, err := os.ReadFile(filepath.FromSlash(checkedInRegionsMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Region workspace entry: %w", err)
	}
	processArgsManifest, err := os.ReadFile(filepath.FromSlash(checkedInProcessArgsManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in process args workspace manifest: %w", err)
	}
	processArgsMain, err := os.ReadFile(filepath.FromSlash(checkedInProcessArgsMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in process args workspace entry: %w", err)
	}
	nativeTestArgsManifest, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestArgsManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test args workspace manifest: %w", err)
	}
	nativeTestArgsMain, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestArgsMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test args workspace entry: %w", err)
	}
	nativeTestArgsTest, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestArgsTestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test args test: %w", err)
	}
	nativeTestStdinManifest, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestStdinManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test stdin workspace manifest: %w", err)
	}
	nativeTestStdinMain, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestStdinMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test stdin workspace entry: %w", err)
	}
	nativeTestStdinTest, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestStdinTestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test stdin test: %w", err)
	}
	nativeTestOutputManifest, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestOutputManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test output workspace manifest: %w", err)
	}
	nativeTestOutputMain, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestOutputMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test output workspace entry: %w", err)
	}
	nativeTestOutputTest, err := os.ReadFile(filepath.FromSlash(checkedInNativeTestOutputTestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in native-test output test: %w", err)
	}
	windowsFileSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsFilePath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows File source: %w", err)
	}
	windowsFileSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsFileSupportPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in Windows File support source: %w", err)
	}
	ownedTextSource, err := os.ReadFile(filepath.FromSlash(checkedInOwnedTextPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in owned-text source: %w", err)
	}
	ownedTextSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInOwnedTextSupportPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in owned-text support source: %w", err)
	}
	workspaceManifest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in workspace manifest: %w", err)
	}
	workspaceMain, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMainPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in workspace entry: %w", err)
	}
	workspaceMemberManifest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMemberManifestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in workspace member manifest: %w", err)
	}
	workspaceMemberSource, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMemberSourcePath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in workspace member source: %w", err)
	}
	workspaceMemberTest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceMemberTestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in workspace member test: %w", err)
	}
	workspaceTest, err := os.ReadFile(filepath.FromSlash(checkedInWorkspaceTestPath))
	if err != nil {
		return nil, fmt.Errorf("read checked-in workspace test: %w", err)
	}
	return map[string]string{
		workspaceManifestPath:                  string(workspaceManifest),
		workspaceMainPath:                      string(workspaceMain),
		workspaceMemberManifestPath:            string(workspaceMemberManifest),
		workspaceMemberSourcePath:              string(workspaceMemberSource),
		workspaceMemberTestPath:                string(workspaceMemberTest),
		workspaceTestPath:                      string(workspaceTest),
		checkedInSuccessPath:                   string(successSource),
		checkedInModuleMainPath:                string(moduleMainSource),
		checkedInModuleSupportPath:             string(moduleSupportSource),
		checkedInWindowsTargetPath:             string(windowsTargetSource),
		checkedInWindowsQuickstartPath:         string(windowsQuickstartSource),
		checkedInWindowsOutputPath:             string(windowsOutputSource),
		checkedInWindowsOutputSupportPath:      string(windowsOutputSupportSource),
		checkedInStdIOManifestPath:             string(stdIOManifest),
		checkedInStdIOMainPath:                 string(stdIOMain),
		checkedInStdIoScalarManifestPath:       string(stdIoScalarManifest),
		checkedInStdIoScalarMainPath:           string(stdIoScalarMain),
		checkedInStdAllocManifestPath:          string(stdAllocManifest),
		checkedInStdAllocMainPath:              string(stdAllocMain),
		checkedInStdBytesManifestPath:          string(stdBytesManifest),
		checkedInStdBytesMainPath:              string(stdBytesMain),
		checkedInStdTextManifestPath:           string(stdTextManifest),
		checkedInStdTextMainPath:               string(stdTextMain),
		checkedInStdTextFromBufferManifestPath: string(stdTextFromBufferManifest),
		checkedInStdTextFromBufferMainPath:     string(stdTextFromBufferMain),
		checkedInStdFSManifestPath:             string(stdFSManifest),
		checkedInStdFSMainPath:                 string(stdFSMain),
		checkedInStdFSPathOpsPath:              string(stdFSPathOps),
		checkedInStdFSPositioningPath:          string(stdFSPositioning),
		checkedInStdFSMetadataPath:             string(stdFSMetadata),
		checkedInStdFSLinkOpsPath:              string(stdFSLinkOps),
		checkedInRegionsManifestPath:           string(regionsManifest),
		checkedInRegionsMainPath:               string(regionsMain),
		checkedInProcessArgsManifestPath:       string(processArgsManifest),
		checkedInProcessArgsMainPath:           string(processArgsMain),
		checkedInNativeTestArgsManifestPath:    string(nativeTestArgsManifest),
		checkedInNativeTestArgsMainPath:        string(nativeTestArgsMain),
		checkedInNativeTestArgsTestPath:        string(nativeTestArgsTest),
		checkedInNativeTestStdinManifestPath:   string(nativeTestStdinManifest),
		checkedInNativeTestStdinMainPath:       string(nativeTestStdinMain),
		checkedInNativeTestStdinTestPath:       string(nativeTestStdinTest),
		checkedInNativeTestOutputManifestPath:  string(nativeTestOutputManifest),
		checkedInNativeTestOutputMainPath:      string(nativeTestOutputMain),
		checkedInNativeTestOutputTestPath:      string(nativeTestOutputTest),
		checkedInWindowsFilePath:               string(windowsFileSource),
		checkedInWindowsFileSupportPath:        string(windowsFileSupportSource),
		checkedInOwnedTextPath:                 string(ownedTextSource),
		checkedInOwnedTextSupportPath:          string(ownedTextSupportSource),
		"negative.fy":                          negativeSource,
		"division_by_zero.fy":                  divisionByZeroSource,
		"remainder_by_zero.fy":                 remainderByZeroSource,
		"signed_division_overflow.fy":          signedDivisionOverflowSource,
		"integer_overflow.fy":                  integerOverflowSource,
		"usize_overflow.fy":                    usizeOverflowSource,
		"unsigned_division.fy":                 unsignedDivisionSource,
		"unsigned_division_zero.fy":            unsignedDivisionZeroSource,
		"unsigned_bitwise_shift.fy":            unsignedBitwiseShiftSource,
		"unsigned_invalid_shift.fy":            unsignedInvalidShiftSource,
		"left_shift.fy":                        leftShiftSource,
		"left_shift_overflow.fy":               leftShiftOverflowSource,
		"invalid_shift.fy":                     invalidShiftSource,
		"bounds.fy":                            boundsSource,
		"array_bounds.fy":                      arrayBoundsSource,
		"array_store_bounds.fy":                arrayStoreBoundsSource,
		"slice_bounds.fy":                      sliceBoundsSource,
		"assertion_failure.fy":                 assertionFailureSource,
		"lexical_defer.fy":                     lexicalDeferSource,
		"enum_result.fy":                       enumResultSource,
		"enum_result_propagation.fy":           enumResultPropagationSource,
		"enum_option.fy":                       enumOptionSource,
		"tagged_array.fy":                      taggedArraySource,
	}, nil
}
