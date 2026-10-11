// Package candidatebuild bridges the Fairy-owned frontend to private native
// transport. It contains no Fairy parser, type checker, IR lowerer or emitter.
package candidatebuild

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const maxPacket = 96 * 1024 * 1024

// Matches candidate_limits::project_frames; C unit count is separately bounded.
const maxProjectFrames = 8192

type Packet struct {
	Header, Literals string
	Units            []string
}

var (
	literalDeclarations = regexp.MustCompile(`extern const uint8_t fy_unit_literal_([0-9a-f]{64})\[\];`)
	literalDefinitions  = regexp.MustCompile(`(?m)^const uint8_t fy_unit_literal_([0-9a-f]{64})\[\]=\{([0-9,]+)\};$`)
	frameDefinitions    = regexp.MustCompile(`(?m)^extern [^\n]+ fy_fn_([0-9]+)\([^\n]*\)\{`)
)

func Decode(data []byte) (Packet, error) {
	if len(data) > maxPacket || !bytes.HasPrefix(data, []byte("FYCPARTS1\n")) {
		return Packet{}, fmt.Errorf("invalid bounded project packet")
	}
	remaining := data[len("FYCPARTS1\n"):]
	number := func() (int, error) {
		end := bytes.IndexByte(remaining, '\n')
		if end < 1 || end > 8 {
			return 0, fmt.Errorf("invalid packet number")
		}
		text := remaining[:end]
		if len(text) > 1 && text[0] == '0' {
			return 0, fmt.Errorf("noncanonical packet number")
		}
		for _, b := range text {
			if b < '0' || b > '9' {
				return 0, fmt.Errorf("invalid packet number")
			}
		}
		n, err := strconv.Atoi(string(text))
		remaining = remaining[end+1:]
		return n, err
	}
	part := func() (string, error) {
		n, err := number()
		if err != nil {
			return "", err
		}
		if n < 1 || n > 2097152 || n > len(remaining) {
			return "", fmt.Errorf("truncated or oversized packet part")
		}
		text := string(remaining[:n])
		remaining = remaining[n:]
		return text, nil
	}
	header, err := part()
	if err != nil {
		return Packet{}, err
	}
	literals, err := part()
	if err != nil {
		return Packet{}, err
	}
	count, err := number()
	if err != nil || count < 1 || count > 4096 {
		return Packet{}, fmt.Errorf("invalid packet unit count")
	}
	units := make([]string, count)
	for i := range units {
		units[i], err = part()
		if err != nil {
			return Packet{}, err
		}
	}
	if len(remaining) != 0 {
		return Packet{}, fmt.Errorf("trailing packet bytes")
	}
	if !strings.HasPrefix(header, "#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n") || !strings.HasSuffix(header, "#endif\n") || !strings.HasPrefix(literals, "/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n") {
		return Packet{}, fmt.Errorf("invalid project C markers")
	}
	declarations := literalDeclarations.FindAllStringSubmatch(header, -1)
	definitions := literalDefinitions.FindAllStringSubmatch(literals, -1)
	if len(declarations) != len(definitions) || strings.Contains(header, "static const uint8_t fy_unit_literal_") {
		return Packet{}, fmt.Errorf("invalid shared literal storage")
	}
	lastDigest := ""
	for i, d := range definitions {
		if d[1] != declarations[i][1] || d[1] <= lastDigest {
			return Packet{}, fmt.Errorf("duplicate or unordered literal identity")
		}
		lastDigest = d[1]
		values := strings.Split(d[2], ",")
		payload := make([]byte, len(values))
		for j, value := range values {
			n, err := strconv.ParseUint(value, 10, 8)
			if err != nil {
				return Packet{}, fmt.Errorf("invalid literal byte")
			}
			payload[j] = byte(n)
		}
		digest := sha256.Sum256(payload)
		empty := sha256.Sum256(nil)
		if hex.EncodeToString(digest[:]) != d[1] && !(len(payload) == 1 && payload[0] == 0 && hex.EncodeToString(empty[:]) == d[1]) {
			return Packet{}, fmt.Errorf("literal identity mismatch")
		}
	}
	lastFrame, entries := 0, 0
	for _, unit := range units {
		if !strings.HasPrefix(unit, "/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\n") || strings.Contains(unit, "const uint8_t fy_unit_literal_") {
			return Packet{}, fmt.Errorf("invalid project body marker or local literal storage")
		}
		for _, match := range frameDefinitions.FindAllStringSubmatch(unit, -1) {
			id, err := strconv.Atoi(match[1])
			if err != nil || id <= lastFrame || id > maxProjectFrames {
				return Packet{}, fmt.Errorf("duplicate or unordered original frame")
			}
			lastFrame = id
		}
		entries += strings.Count(unit, "int32_t fy_candidate_entry(void){")
	}
	if lastFrame == 0 || entries != 1 {
		return Packet{}, fmt.Errorf("project packet requires one entry")
	}
	return Packet{header, literals, units}, nil
}
