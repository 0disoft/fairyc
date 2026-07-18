package semantic

import "strings"

func parseOptionType(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	const prefix = "Option<"
	if len(trimmed) <= len(prefix)+1 || !strings.HasPrefix(trimmed, prefix) || trimmed[len(trimmed)-1] != '>' {
		return "", false
	}
	inside := strings.TrimSpace(trimmed[len(prefix) : len(trimmed)-1])
	if inside == "" {
		return "", false
	}
	depth := 0
	for index := 0; index < len(inside); index++ {
		switch inside[index] {
		case '<':
			depth++
		case '>':
			depth--
			if depth < 0 {
				return "", false
			}
		case ',':
			if depth == 0 {
				return "", false
			}
		}
	}
	return inside, depth == 0
}
