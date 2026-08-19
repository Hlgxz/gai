package orm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type timeScanner struct {
	t *time.Time
}

func (s *timeScanner) Scan(src any) error {
	tm, err := parseDBTime(src)
	if err != nil {
		return err
	}
	if tm != nil {
		*s.t = *tm
	}
	return nil
}

type nullTimeScanner struct {
	t **time.Time
}

func (s *nullTimeScanner) Scan(src any) error {
	if src == nil {
		*s.t = nil
		return nil
	}
	if b, ok := src.([]byte); ok && len(b) == 0 {
		*s.t = nil
		return nil
	}
	if str, ok := src.(string); ok && str == "" {
		*s.t = nil
		return nil
	}
	tm, err := parseDBTime(src)
	if err != nil {
		return err
	}
	*s.t = tm
	return nil
}

func parseDBTime(src any) (*time.Time, error) {
	switch v := src.(type) {
	case time.Time:
		return &v, nil
	case string:
		return parseTimeString(v)
	case []byte:
		return parseTimeString(string(v))
	case int64:
		t := time.Unix(v, 0)
		return &t, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("gai/orm: cannot scan %T into time.Time", src)
	}
}

func parseTimeString(s string) (*time.Time, error) {
	if i := strings.Index(s, " m="); i >= 0 {
		s = s[:i]
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.RFC1123Z,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return &t, nil
		}
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("gai/orm: cannot parse time %q", s)
}

func wrapScanDest(ptr any) any {
	switch p := ptr.(type) {
	case *time.Time:
		return &timeScanner{t: p}
	case **time.Time:
		return &nullTimeScanner{t: p}
	default:
		rv := reflect.ValueOf(ptr)
		if rv.Kind() == reflect.Ptr && rv.Elem().Type() == reflect.TypeOf(time.Time{}) {
			return &timeScanner{t: ptr.(*time.Time)}
		}
		return ptr
	}
}

func bindArg(v any) any {
	switch t := v.(type) {
	case time.Time:
		return t.UTC().Format(time.RFC3339Nano)
	case *time.Time:
		if t == nil {
			return nil
		}
		return t.UTC().Format(time.RFC3339Nano)
	default:
		return v
	}
}
