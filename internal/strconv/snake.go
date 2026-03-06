package strconv

import (
	"strings"
	"unicode"
)

func ToSnakeCaseSelf(s string) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}

	var words []string
	wordStart := 0

	for i := 1; i < n; i++ {
		prev, cur := runes[i-1], runes[i]

		boundary := false

		switch {
		case unicode.IsLower(prev) && unicode.IsUpper(cur):
			boundary = true
		case unicode.IsUpper(prev) && unicode.IsUpper(cur) && i+1 < n && unicode.IsLower(runes[i+1]):
			boundary = true
		case unicode.IsDigit(prev) && unicode.IsUpper(cur):
			boundary = true
		}

		if boundary {
			words = append(words, string(runes[wordStart:i]))
			wordStart = i
		}
	}

	words = append(words, string(runes[wordStart:]))

	for i, w := range words {
		words[i] = strings.ToLower(w)
	}

	return strings.Join(words, "_")
}

func ToSnakeCase(s string) string {
	return ToSnakeCaseSelf(s)
}
