package lexer

import "slices"

// EditorVocabulary returns copies of the current lexer vocabulary for editor assets.
func EditorVocabulary() (keywords, operators []string) {
	for word := range currentKeywords {
		keywords = append(keywords, word)
	}
	for word := range currentRevisionOnlyKeywords {
		keywords = append(keywords, word)
	}
	slices.Sort(keywords)
	return keywords, slices.Clone(symbols)
}
