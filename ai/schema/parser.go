package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Hlgxz/gai/support"
)

// ParseFile reads and parses a single YAML schema file.
func ParseFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gai/schema: cannot read %s: %w", path, err)
	}

	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("gai/schema: invalid YAML in %s: %w", path, err)
	}

	if s.Model == "" {
		base := filepath.Base(path)
		s.Model = support.Camel(strings.TrimSuffix(base, filepath.Ext(base)))
	}

	if s.Table == "" {
		s.Table = strings.ToLower(support.Plural(support.Snake(s.Model)))
	}

	return &s, nil
}

// ParseDir reads all .yaml/.yml files in a directory and returns a slice of schemas.
func ParseDir(dir string) ([]*Schema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("gai/schema: cannot read directory %s: %w", dir, err)
	}

	var schemas []*Schema
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		s, err := ParseFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, s)
	}

	return schemas, nil
}
