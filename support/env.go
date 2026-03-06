package support

import (
	"os"
	"strconv"
	"strings"
)

// Env returns the value of the environment variable named by key,
// or the first fallback value if the variable is not set.
func Env(key string, fallback ...string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// EnvInt returns the integer value of the environment variable named by key,
// or the first fallback value if the variable is not set or not a valid integer.
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

// EnvBool returns the boolean value of the environment variable named by key.
// It recognises "true", "1", "yes", "on" as true and "false", "0", "no", "off"
// as false. Returns the first fallback if the variable is unset or unrecognised.
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
