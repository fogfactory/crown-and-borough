package assetgen

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Slug lowercases a name, drops its accents and joins its words with hyphens
// ("Adélaïde" -> "adelaide"), for readable seeds and file names.
func Slug(value string) string {
	value = norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var builder strings.Builder
	pendingHyphen := false
	for _, character := range value {
		if unicode.Is(unicode.Mn, character) {
			continue
		}
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if pendingHyphen && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(character)
			pendingHyphen = false
			continue
		}
		if builder.Len() > 0 {
			pendingHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
