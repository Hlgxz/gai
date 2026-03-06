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

// Plural returns a naive English plural form.
func Plural(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "sh") || strings.HasSuffix(lower, "ch") {
		return s + "es"
	}
	if strings.HasSuffix(lower, "y") && len(s) > 1 {
		prev := rune(lower[len(lower)-2])
		if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}
