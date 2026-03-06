package config

import (
	"bufio"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadYAMLFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// LoadEnvFile reads a .env file and sets environment variables that are not
// already defined, so real env vars always take precedence.
func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}
