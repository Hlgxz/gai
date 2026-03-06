package support

import (
	"os"
	"strconv"
	"strings"
)

func Env(key string, fallback ...string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

func EnvInt(key string, fallback ...int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

func EnvBool(key string, fallback ...bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		switch strings.ToLower(val) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return false
}
