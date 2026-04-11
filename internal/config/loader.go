package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a YAML config file. If the file does not exist,
// a default config is created.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := Save(path, cfg); err != nil {
			return nil, fmt.Errorf("create default config: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// Save writes the config as YAML to the given path.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ParseSize parses a human-readable size such as "10MB", "2GB", "512KB", or "10MB/min".
// Returns the value in bytes (or bytes per minute when /min suffix is used).
func ParseSize(s string) uint64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	// Strip "/MIN" suffix if present.
	s = strings.TrimSuffix(s, "/MIN")

	multipliers := []struct {
		suffix string
		mult   uint64
	}{
		{"TB", 1 << 40},
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			n := strings.TrimSuffix(s, m.suffix)
			n = strings.TrimSpace(n)
			val, err := strconv.ParseFloat(n, 64)
			if err != nil {
				return 0
			}
			return uint64(val * float64(m.mult))
		}
	}
	val, _ := strconv.ParseUint(s, 10, 64)
	return val
}
