package http

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// RowExister is implemented by *orm.DB (ExistsRow) for unique/exists rules.
type RowExister interface {
	ExistsRow(table, column string, value any) (bool, error)
}

// WithDB enables unique:table,column and exists:table,column rules.
func (v *Validator) WithDB(db RowExister) *Validator {
	v.store = db
	return v
}

func (v *Validator) applyRule(field string, value any, exists bool, ruleName, ruleParam string) {
	switch ruleName {
	case "confirmed":
		other, _ := toString(v.data[field+"_confirmation"])
		s, _ := toString(value)
		if s != other {
			v.addError(field, fmt.Sprintf("%s confirmation does not match", field))
		}
	case "same":
		other, _ := toString(v.data[ruleParam])
		s, _ := toString(value)
		if s != other {
			v.addError(field, fmt.Sprintf("%s must match %s", field, ruleParam))
		}
	case "integer":
		if s, ok := toString(value); ok {
			if _, err := strconv.Atoi(s); err != nil {
				v.addError(field, fmt.Sprintf("%s must be an integer", field))
			}
		}
	case "boolean":
		switch value.(type) {
		case bool:
		default:
			s, _ := toString(value)
			if s != "true" && s != "false" && s != "1" && s != "0" {
				v.addError(field, fmt.Sprintf("%s must be a boolean", field))
			}
		}
	case "date":
		s, ok := toString(value)
		if !ok || !isDate(s) {
			v.addError(field, fmt.Sprintf("%s must be a valid date", field))
		}
	case "json":
		s, ok := toString(value)
		if !ok || !json.Valid([]byte(s)) {
			v.addError(field, fmt.Sprintf("%s must be valid JSON", field))
		}
	case "ip":
		s, _ := toString(value)
		if net.ParseIP(s) == nil {
			v.addError(field, fmt.Sprintf("%s must be a valid IP address", field))
		}
	case "uuid":
		s, _ := toString(value)
		if !isUUID(s) {
			v.addError(field, fmt.Sprintf("%s must be a valid UUID", field))
		}
	case "unique":
		v.checkExists(field, value, ruleParam, true)
	case "exists":
		v.checkExists(field, value, ruleParam, false)
	case "required_if":
		parts := strings.SplitN(ruleParam, ",", 2)
		if len(parts) != 2 {
			return
		}
		other, _ := toString(v.data[parts[0]])
		if other == parts[1] && (!exists || isEmpty(value)) {
			v.addError(field, fmt.Sprintf("%s is required", field))
		}
	}
}

func (v *Validator) checkExists(field string, value any, param string, wantUnique bool) {
	if v.store == nil {
		return
	}
	parts := strings.Split(param, ",")
	if len(parts) < 2 {
		return
	}
	table, column := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	ok, err := v.store.ExistsRow(table, column, value)
	if err != nil {
		v.addError(field, fmt.Sprintf("%s validation query failed", field))
		return
	}
	if wantUnique && ok {
		v.addError(field, fmt.Sprintf("%s has already been taken", field))
	}
	if !wantUnique && !ok {
		v.addError(field, fmt.Sprintf("%s does not exist", field))
	}
}

func isDate(s string) bool {
	layouts := []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"}
	for _, l := range layouts {
		if _, err := time.Parse(l, s); err == nil {
			return true
		}
	}
	return false
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
