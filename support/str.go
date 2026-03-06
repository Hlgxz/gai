package support

import (
	"strings"
	"unicode"
)

// Snake converts CamelCase to snake_case.
func Snake(s string) string {
	var buf strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				buf.WriteByte('_')
			}
			buf.WriteRune(unicode.ToLower(r))
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// Camel converts snake_case to CamelCase.
func Camel(s string) string {
	var buf strings.Builder
	upper := true
	for _, r := range s {
		if r == '_' || r == '-' {
			upper = true
			continue
		}
		if upper {
			buf.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// LowerCamel converts snake_case to lowerCamelCase.
func LowerCamel(s string) string {
	c := Camel(s)
	if len(c) == 0 {
		return c
	}
	return strings.ToLower(c[:1]) + c[1:]
}

var irregularPlurals = map[string]string{
	"person": "people", "child": "children", "man": "men", "woman": "women",
	"mouse": "mice", "goose": "geese", "tooth": "teeth", "foot": "feet",
	"ox": "oxen", "leaf": "leaves", "life": "lives", "knife": "knives",
	"wife": "wives", "half": "halves", "self": "selves", "shelf": "shelves",
	"series": "series", "species": "species", "deer": "deer", "sheep": "sheep",
	"fish": "fish", "datum": "data", "index": "indices", "matrix": "matrices",
}

// Plural returns an English plural form, handling common irregular nouns
// and standard suffix rules.
func Plural(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)

	if plural, ok := irregularPlurals[lower]; ok {
		if s[0] == lower[0] {
			return plural
		}
		return strings.ToUpper(plural[:1]) + plural[1:]
	}

	if strings.HasSuffix(lower, "ss") || strings.HasSuffix(lower, "sh") ||
		strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "z") || strings.HasSuffix(lower, "o") {
		return s + "es"
	}
	if strings.HasSuffix(lower, "s") {
		return s + "es"
	}
	if strings.HasSuffix(lower, "fe") {
		return s[:len(s)-2] + "ves"
	}
	if strings.HasSuffix(lower, "f") {
		return s[:len(s)-1] + "ves"
	}
	if strings.HasSuffix(lower, "y") {
		runes := []rune(lower)
		if len(runes) > 1 {
			prev := runes[len(runes)-2]
			if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
				return string([]rune(s)[:len(runes)-1]) + "ies"
			}
		}
	}
	return s + "s"
}
