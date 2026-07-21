package prevrevision

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	sourceRootPattern = regexp.MustCompile(`"sourceRoot"\s*:\s*"([^"]+)"`)
	caseIDPattern     = regexp.MustCompile(`"id"\s*:\s*"([^"]+)"`)
	spanPattern       = regexp.MustCompile(`(?s)\{\s*"source"\s*:\s*"([^"]*)"\s*,\s*"startByte"\s*:\s*([0-9]+)\s*,\s*"endByte"\s*:\s*([0-9]+)\s*\}`)
)

// ManifestReport describes a fail-closed fixture-span refresh.
type ManifestReport struct {
	OldRef    string   `json:"oldRef"`
	Mode      string   `json:"mode"`
	Manifests int      `json:"manifests"`
	Spans     int      `json:"spans"`
	Overrides int      `json:"nonIsomorphicSpans"`
	Written   int      `json:"written"`
	Files     []string `json:"files,omitempty"`
}

type plannedManifest struct {
	path   string
	rel    string
	mode   os.FileMode
	source []byte
}

// RefreshFixtureSpans maps manifest spans through the exact source migration committed after oldRef.
func RefreshFixtureSpans(repoRoot, oldRef string, manifests []string, write bool) (ManifestReport, error) {
	absoluteRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return ManifestReport{}, err
	}
	report := ManifestReport{OldRef: oldRef, Mode: "check"}
	if write {
		report.Mode = "write"
	}
	mapperCache := make(map[string]SpanMapper)
	var plan []plannedManifest
	for _, relativeManifest := range manifests {
		manifestPath := filepath.Join(absoluteRoot, filepath.FromSlash(relativeManifest))
		current, err := os.ReadFile(manifestPath)
		if err != nil {
			return report, fmt.Errorf("read %s: %w", relativeManifest, err)
		}
		baseline, err := gitShow(absoluteRoot, oldRef, filepath.ToSlash(relativeManifest))
		if err != nil {
			return report, err
		}
		currentShape, err := manifestWithoutSpanNumbers(current)
		if err != nil {
			return report, fmt.Errorf("inspect current %s: %w", relativeManifest, err)
		}
		baselineShape, err := manifestWithoutSpanNumbers(baseline)
		if err != nil {
			return report, fmt.Errorf("inspect baseline %s: %w", relativeManifest, err)
		}
		if !bytes.Equal(currentShape, baselineShape) {
			return report, fmt.Errorf("current %s differs from %s outside source-span numbers", relativeManifest, oldRef)
		}
		output, count, overrides, err := refreshManifestSpans(absoluteRoot, oldRef, manifestPath, baseline, mapperCache)
		if err != nil {
			return report, fmt.Errorf("refresh %s: %w", relativeManifest, err)
		}
		report.Manifests++
		report.Spans += count
		report.Overrides += overrides
		if bytes.Equal(current, output) {
			continue
		}
		info, err := os.Stat(manifestPath)
		if err != nil {
			return report, err
		}
		plan = append(plan, plannedManifest{path: manifestPath, rel: filepath.ToSlash(relativeManifest), mode: info.Mode(), source: output})
	}
	slices.SortFunc(plan, func(left, right plannedManifest) int { return strings.Compare(left.rel, right.rel) })
	for _, file := range plan {
		report.Files = append(report.Files, file.rel)
	}
	if !write {
		return report, nil
	}
	for _, file := range plan {
		if err := atomicReplace(file.path, file.source, file.mode); err != nil {
			return report, fmt.Errorf("write %s: %w", file.rel, err)
		}
		report.Written++
	}
	return report, nil
}

func refreshManifestSpans(repoRoot, oldRef, manifestPath string, input []byte, cache map[string]SpanMapper) ([]byte, int, int, error) {
	cases, err := caseObjectRanges(input)
	if err != nil {
		return nil, 0, 0, err
	}
	manifestDirectory := filepath.Dir(manifestPath)
	var edits []edit
	spanCount := 0
	overrideCount := 0
	for _, caseRange := range cases {
		caseBytes := input[caseRange.start:caseRange.end]
		idMatch := caseIDPattern.FindSubmatch(caseBytes)
		if idMatch == nil {
			return nil, 0, 0, fmt.Errorf("fixture case has no id")
		}
		caseID := string(idMatch[1])
		rootMatch := sourceRootPattern.FindSubmatch(caseBytes)
		if rootMatch == nil {
			continue
		}
		sourceRoot := string(rootMatch[1])
		matches := spanPattern.FindAllSubmatchIndex(caseBytes, -1)
		for _, match := range matches {
			sourceName := string(caseBytes[match[2]:match[3]])
			if sourceName == "" {
				continue
			}
			start, _ := strconv.Atoi(string(caseBytes[match[4]:match[5]]))
			end, _ := strconv.Atoi(string(caseBytes[match[6]:match[7]]))
			sourcePath := filepath.Clean(filepath.Join(manifestDirectory, filepath.FromSlash(sourceRoot), filepath.FromSlash(sourceName)))
			repoRelative, err := filepath.Rel(repoRoot, sourcePath)
			if err != nil || strings.HasPrefix(repoRelative, "..") {
				return nil, 0, 0, fmt.Errorf("source %s escapes repository", sourceName)
			}
			repoRelative = filepath.ToSlash(repoRelative)
			mapper, found := cache[repoRelative]
			if !found {
				oldSource, err := gitShow(repoRoot, oldRef, repoRelative)
				if err != nil {
					return nil, 0, 0, err
				}
				currentSource, err := os.ReadFile(sourcePath)
				if err != nil {
					return nil, 0, 0, err
				}
				migrated, built, err := BuildSpanMapper(sourceName, oldSource)
				if err != nil {
					return nil, 0, 0, fmt.Errorf("map %s: %w", repoRelative, err)
				}
				if !bytes.Equal(migrated, currentSource) {
					return nil, 0, 0, fmt.Errorf("migrated %s does not equal current source", repoRelative)
				}
				mapper = built
				cache[repoRelative] = mapper
			}
			mappedStart, mappedEnd, overridden := nonIsomorphicSpan(manifestPath, caseID, sourceName, start, end)
			if overridden {
				overrideCount++
			} else {
				mappedStart, mappedEnd, err = mapper.Map(start, end)
				if err != nil {
					return nil, 0, 0, fmt.Errorf("map %s span %d..%d: %w", repoRelative, start, end, err)
				}
			}
			absoluteStartNumber := caseRange.start + match[4]
			absoluteEndNumber := caseRange.start + match[6]
			edits = append(edits,
				edit{start: absoluteStartNumber, end: caseRange.start + match[5], replacement: strconv.Itoa(mappedStart)},
				edit{start: absoluteEndNumber, end: caseRange.start + match[7], replacement: strconv.Itoa(mappedEnd)},
			)
			spanCount++
		}
	}
	output, err := applyEdits(input, edits)
	return output, spanCount, overrideCount, err
}

// nonIsomorphicSpan records diagnostics whose owning AST node intentionally became narrower when
// C-shaped declarations and explicit returns replaced the previous surface forms. Byte-edit
// mapping cannot derive these spans because the semantic node itself changed.
func nonIsomorphicSpan(manifestPath, caseID, sourceName string, start, end int) (int, int, bool) {
	manifest := filepath.ToSlash(manifestPath)
	switch {
	case strings.Contains(manifest, "/result/") && caseID == "fail.type.incompatible-error" &&
		sourceName == "main.fy" && start == 188 && end == 199:
		return 97, 108, true
	case strings.Contains(manifest, "/semantic/") && caseID == "fail.type.foreign-safe-reference" &&
		sourceName == "main.fy" && start == 48 && end == 52:
		return 45, 52, true
	case strings.Contains(manifest, "/semantic/") && caseID == "fail.flow.missing-resource-drop-effect" &&
		sourceName == "main.fy" && start == 164 && end == 217:
		return 193, 202, true
	default:
		return 0, 0, false
	}
}

func manifestWithoutSpanNumbers(input []byte) ([]byte, error) {
	var edits []edit
	for _, match := range spanPattern.FindAllSubmatchIndex(input, -1) {
		edits = append(edits,
			edit{start: match[4], end: match[5], replacement: "#"},
			edit{start: match[6], end: match[7], replacement: "#"},
		)
	}
	return applyEdits(input, edits)
}

type byteRange struct{ start, end int }

func caseObjectRanges(input []byte) ([]byteRange, error) {
	key := bytes.Index(input, []byte(`"cases"`))
	if key < 0 {
		return nil, nil
	}
	arrayOffset := bytes.IndexByte(input[key:], '[')
	if arrayOffset < 0 {
		return nil, fmt.Errorf("cases has no array")
	}
	arrayStart := key + arrayOffset
	inString, escaped := false, false
	arrayDepth, objectDepth, objectStart := 0, 0, -1
	var result []byteRange
	for index := arrayStart; index < len(input); index++ {
		value := input[index]
		if inString {
			if escaped {
				escaped = false
			} else if value == '\\' {
				escaped = true
			} else if value == '"' {
				inString = false
			}
			continue
		}
		if value == '"' {
			inString = true
			continue
		}
		switch value {
		case '[':
			arrayDepth++
		case ']':
			arrayDepth--
			if arrayDepth == 0 {
				return result, nil
			}
		case '{':
			if arrayDepth == 1 && objectDepth == 0 {
				objectStart = index
			}
			objectDepth++
		case '}':
			objectDepth--
			if arrayDepth == 1 && objectDepth == 0 && objectStart >= 0 {
				result = append(result, byteRange{start: objectStart, end: index + 1})
				objectStart = -1
			}
		}
	}
	return nil, fmt.Errorf("unterminated cases array")
}

func gitShow(repoRoot, oldRef, relative string) ([]byte, error) {
	command := exec.Command("git", "-C", repoRoot, "show", oldRef+":"+relative)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w", oldRef, relative, err)
	}
	return output, nil
}
