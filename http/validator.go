package http

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	reAlpha        = regexp.MustCompile(`^[a-zA-Z]+$`)
	reAlphanumeric = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	rePhone        = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

// ValidationErrors maps field names to their error messages.
type ValidationErrors map[string][]string

func (ve ValidationErrors) Error() string {
	var parts []string
	for field, msgs := range ve {
		for _, msg := range msgs {
			parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
		}
	}
	return strings.Join(parts, "; ")
}

// HasErrors returns true if there are validation failures.
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// First returns the first error message for the given field.
func (ve ValidationErrors) First(field string) string {
	if msgs, ok := ve[field]; ok && len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

// Validator validates a map of field values against rules.
// Rules use Laravel-style pipe-separated syntax: "required|email|min:5".
type Validator struct {
	data   map[string]any
	rules  map[string]string
	errors ValidationErrors
}

// NewValidator creates a validator for the given data and rules.
func NewValidator(data map[string]any, rules map[string]string) *Validator {
	return &Validator{
		data:   data,
		rules:  rules,
		errors: make(ValidationErrors),
	}
}

// Validate runs all rules and returns errors (nil if valid).
func (v *Validator) Validate() ValidationErrors {
	for field, ruleStr := range v.rules {
		value, exists := v.data[field]
		rules := strings.Split(ruleStr, "|")

		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}

			ruleName, ruleParam := parseRule(rule)

			if ruleName == "required" {
				if !exists || isEmpty(value) {
					v.addError(field, fmt.Sprintf("%s is required", field))
				}
				continue
			}

			if !exists || isEmpty(value) {
				continue
			}

			switch ruleName {
			case "email":
				if s, ok := toString(value); ok {
					if _, err := mail.ParseAddress(s); err != nil {
						v.addError(field, fmt.Sprintf("%s must be a valid email", field))
					}
				}
			case "min":
				v.validateMin(field, value, ruleParam)
			case "max":
				v.validateMax(field, value, ruleParam)
			case "numeric":
				if s, ok := toString(value); ok {
					if _, err := strconv.ParseFloat(s, 64); err != nil {
						v.addError(field, fmt.Sprintf("%s must be numeric", field))
					}
				}
			case "alpha":
				if s, ok := toString(value); ok {
					if !reAlpha.MatchString(s) {
						v.addError(field, fmt.Sprintf("%s must contain only letters", field))
					}
				}
			case "alphanumeric":
				if s, ok := toString(value); ok {
					if !reAlphanumeric.MatchString(s) {
						v.addError(field, fmt.Sprintf("%s must be alphanumeric", field))
					}
				}
			case "phone":
				if s, ok := toString(value); ok {
					if !rePhone.MatchString(s) {
						v.addError(field, fmt.Sprintf("%s must be a valid phone number", field))
					}
				}
			case "url":
				if s, ok := toString(value); ok {
					u, err := url.Parse(s)
					if err != nil || u.Scheme == "" || u.Host == "" {
						v.addError(field, fmt.Sprintf("%s must be a valid URL", field))
					}
				}
			case "in":
				if s, ok := toString(value); ok {
					allowed := strings.Split(ruleParam, ",")
					found := false
					for _, a := range allowed {
						if strings.TrimSpace(a) == s {
							found = true
							break
						}
					}
					if !found {
						v.addError(field, fmt.Sprintf("%s must be one of: %s", field, ruleParam))
					}
				}
			case "regex":
				if s, ok := toString(value); ok {
					re, err := regexp.Compile(ruleParam)
					if err == nil && !re.MatchString(s) {
						v.addError(field, fmt.Sprintf("%s format is invalid", field))
					}
				}
			}
		}
	}

	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

func (v *Validator) addError(field, message string) {
	v.errors[field] = append(v.errors[field], message)
}

func (v *Validator) validateMin(field string, value any, param string) {
	n, _ := strconv.Atoi(param)
	switch val := value.(type) {
	case string:
		if utf8.RuneCountInString(val) < n {
			v.addError(field, fmt.Sprintf("%s must be at least %d characters", field, n))
		}
	case float64:
		if val < float64(n) {
			v.addError(field, fmt.Sprintf("%s must be at least %d", field, n))
		}
	case int:
		if val < n {
			v.addError(field, fmt.Sprintf("%s must be at least %d", field, n))
		}
	}
}

func (v *Validator) validateMax(field string, value any, param string) {
	n, _ := strconv.Atoi(param)
	switch val := value.(type) {
	case string:
		if utf8.RuneCountInString(val) > n {
			v.addError(field, fmt.Sprintf("%s must be at most %d characters", field, n))
		}
	case float64:
		if val > float64(n) {
			v.addError(field, fmt.Sprintf("%s must be at most %d", field, n))
		}
	case int:
		if val > n {
			v.addError(field, fmt.Sprintf("%s must be at most %d", field, n))
		}
	}
}

func parseRule(rule string) (string, string) {
	idx := strings.IndexByte(rule, ':')
	if idx < 0 {
		return rule, ""
	}
	return rule[:idx], rule[idx+1:]
}

func toString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return "", false
	}
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}
