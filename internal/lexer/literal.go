package lexer

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/diagnostic"
)

func (s *scanner) scanNumber() {
	start := s.position
	kind := Integer
	if s.matchAt(start, "0x") || s.matchAt(start, "0b") {
		s.position += 2
		for s.position < len(s.source) && isIdentifierContinue(s.source[s.position]) {
			s.position++
		}
	} else {
		for s.position < len(s.source) && (isDecimalDigit(s.source[s.position]) || s.source[s.position] == '_') {
			s.position++
		}
		if s.position+1 < len(s.source) && s.source[s.position] == '.' && isDecimalDigit(s.source[s.position+1]) {
			kind = Float
			s.position += 2
			for s.position < len(s.source) && (isDecimalDigit(s.source[s.position]) || s.source[s.position] == '_') {
				s.position++
			}
		}
		if s.position < len(s.source) && (s.source[s.position] == 'e' || s.source[s.position] == 'E') {
			kind = Float
			s.position++
			if s.position < len(s.source) && (s.source[s.position] == '+' || s.source[s.position] == '-') {
				s.position++
			}
			for s.position < len(s.source) && (isDecimalDigit(s.source[s.position]) || s.source[s.position] == '_') {
				s.position++
			}
		}
		for s.position < len(s.source) && isIdentifierContinue(s.source[s.position]) {
			s.position++
		}
	}
	raw := string(s.source[start:s.position])
	cooked, valid := cookNumber(raw, kind)
	s.emit(kind, start, s.position, cooked)
	if !valid {
		s.emitDiagnostic("E0004", start, s.position, diagnostic.RepairNone, "")
	}
}

func cookNumber(raw string, kind Kind) (string, bool) {
	if kind == Integer {
		base := 10
		digits := raw
		validDigit := isDecimalDigit
		if strings.HasPrefix(raw, "0x") {
			base = 16
			digits = raw[2:]
			validDigit = isHexDigit
		} else if strings.HasPrefix(raw, "0b") {
			base = 2
			digits = raw[2:]
			validDigit = isBinaryDigit
		}
		if !validSeparatedDigits(digits, validDigit) {
			return "", false
		}
		value := new(big.Int)
		if _, ok := value.SetString(strings.ReplaceAll(digits, "_", ""), base); !ok {
			return "", false
		}
		return value.String(), true
	}
	return cookDecimal(raw)
}

func validSeparatedDigits(value string, validDigit func(byte) bool) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		current := value[index]
		if validDigit(current) {
			continue
		}
		if current != '_' || index == 0 || index+1 >= len(value) || !validDigit(value[index-1]) || !validDigit(value[index+1]) {
			return false
		}
	}
	return true
}

func cookDecimal(raw string) (string, bool) {
	value := strings.ReplaceAll(raw, "_", "")
	exponent := 0
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		parsed, err := strconv.Atoi(value[index+1:])
		if err != nil {
			return "", false
		}
		exponent = parsed
		value = value[:index]
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return "", false
	}
	if !allDigits(parts[0]) {
		return "", false
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if fraction == "" || !allDigits(fraction) {
			return "", false
		}
	} else if !strings.ContainsAny(raw, "eE") {
		return "", false
	}
	digits := strings.TrimLeft(parts[0]+fraction, "0")
	if digits == "" {
		digits = "0"
	}
	decimalPlaces := len(fraction) - exponent
	var cooked string
	switch {
	case decimalPlaces <= 0:
		cooked = digits + strings.Repeat("0", -decimalPlaces)
	case decimalPlaces >= len(digits):
		cooked = "0." + strings.Repeat("0", decimalPlaces-len(digits)) + digits
	default:
		split := len(digits) - decimalPlaces
		cooked = digits[:split] + "." + digits[split:]
	}
	if strings.Contains(cooked, ".") {
		cooked = strings.TrimRight(cooked, "0")
		cooked = strings.TrimSuffix(cooked, ".")
	}
	return cooked, true
}

func allDigits(value string) bool {
	for index := 0; index < len(value); index++ {
		if !isDecimalDigit(value[index]) {
			return false
		}
	}
	return value != ""
}

func (s *scanner) scanString(kind Kind, openerBytes int) {
	start := s.position
	s.position += openerBytes
	cooked := make([]byte, 0, 32)
	terminated := false
	for s.position < len(s.source) {
		current := s.source[s.position]
		if current == '"' {
			s.position++
			terminated = true
			break
		}
		if current == '\n' || current == '\r' {
			s.emitDiagnostic("E0005", s.position, s.position+1, diagnostic.RepairExit, "\"")
			break
		}
		if current == '\\' {
			s.scanEscape(kind, &cooked)
			continue
		}
		if current == 0 {
			escapeStart := s.position
			s.position++
			if kind == CString {
				s.emitDiagnostic("E0006", escapeStart, s.position, diagnostic.RepairNone, "")
			} else {
				s.emitDiagnostic("E0005", escapeStart, s.position, diagnostic.RepairNone, "")
			}
			cooked = append(cooked, 0)
			continue
		}
		if current >= utf8.RuneSelf {
			runeValue, width := utf8.DecodeRune(s.source[s.position:])
			if runeValue == utf8.RuneError && width == 1 {
				invalidStart := s.position
				s.position++
				s.emitDiagnostic("E0001", invalidStart, s.position, diagnostic.RepairNone, "")
				continue
			}
			cooked = append(cooked, s.source[s.position:s.position+width]...)
			s.position += width
			continue
		}
		cooked = append(cooked, current)
		s.position++
	}
	if !terminated && s.position >= len(s.source) {
		s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairExit, "\"")
	}
	if kind == CString {
		cooked = append(cooked, 0)
	}
	s.emit(kind, start, s.position, hex.EncodeToString(cooked))
}

func (s *scanner) scanEscape(kind Kind, cooked *[]byte) {
	start := s.position
	s.position++
	if s.position >= len(s.source) {
		s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairNone, "")
		return
	}
	current := s.source[s.position]
	s.position++
	switch current {
	case '\\', '"':
		*cooked = append(*cooked, current)
	case 'n':
		*cooked = append(*cooked, '\n')
	case 'r':
		*cooked = append(*cooked, '\r')
	case 't':
		*cooked = append(*cooked, '\t')
	case '0':
		*cooked = append(*cooked, 0)
		if kind == CString {
			s.emitDiagnostic("E0006", start, s.position, diagnostic.RepairNone, "")
		}
	case 'x':
		if s.position+2 > len(s.source) || !isHexDigit(s.source[s.position]) || !isHexDigit(s.source[s.position+1]) {
			for s.position < len(s.source) && s.position < start+4 && isHexDigit(s.source[s.position]) {
				s.position++
			}
			s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairNone, "")
			return
		}
		value, _ := strconv.ParseUint(string(s.source[s.position:s.position+2]), 16, 8)
		s.position += 2
		*cooked = append(*cooked, byte(value))
		if kind == CString && value == 0 {
			s.emitDiagnostic("E0006", start, s.position, diagnostic.RepairNone, "")
		}
	case 'u':
		if kind != String || s.position >= len(s.source) || s.source[s.position] != '{' {
			s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairNone, "")
			return
		}
		s.position++
		digitsStart := s.position
		for s.position < len(s.source) && isHexDigit(s.source[s.position]) && s.position-digitsStart < 6 {
			s.position++
		}
		if digitsStart == s.position || s.position >= len(s.source) || s.source[s.position] != '}' {
			for s.position < len(s.source) && s.source[s.position] != '}' && s.source[s.position] != '"' {
				s.position++
			}
			if s.position < len(s.source) && s.source[s.position] == '}' {
				s.position++
			}
			s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairNone, "")
			return
		}
		value, err := strconv.ParseUint(string(s.source[digitsStart:s.position]), 16, 32)
		s.position++
		runeValue := rune(value)
		if err != nil || !utf8.ValidRune(runeValue) || runeValue >= 0xd800 && runeValue <= 0xdfff {
			s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairNone, "")
			return
		}
		buffer := make([]byte, utf8.RuneLen(runeValue))
		utf8.EncodeRune(buffer, runeValue)
		*cooked = append(*cooked, buffer...)
	default:
		s.emitDiagnostic("E0005", start, s.position, diagnostic.RepairNone, "")
	}
}

func isBinaryDigit(value byte) bool {
	return value == '0' || value == '1'
}

func isHexDigit(value byte) bool {
	return isDecimalDigit(value) || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}
