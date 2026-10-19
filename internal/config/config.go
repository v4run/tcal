// Package config loads tcal's optional TOML config file.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/BurntSushi/toml"
)

// Config is the parsed config file. Validation of flag values is left to the
// caller, which owns the parsers.
type Config struct {
	Path       string
	Flags      map[string]string // flag name -> value formatted with fmt.Sprint
	Birthday   string
	RandomPool []string // nil when unset
}

// DefaultPath returns $XDG_CONFIG_HOME/tcal/config.toml, falling back to
// ~/.config/tcal/config.toml on every platform.
func DefaultPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "tcal", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tcal", "config.toml")
}

// Load reads path. A missing file yields an empty Config unless mustExist.
// Keys other than birthday, [random] and allowedFlags are errors.
func Load(path string, mustExist bool, allowedFlags []string) (Config, error) {
	cfg := Config{Path: path, Flags: map[string]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !mustExist {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	raw := map[string]any{}
	if _, err := toml.Decode(string(data), &raw); err != nil {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := raw[k]
		switch k {
		case "birthday":
			s, ok := v.(string)
			if !ok {
				return Config{}, fmt.Errorf("config %s: birthday must be a string, got %v", path, v)
			}
			cfg.Birthday = s
		case "random":
			if err := loadRandom(&cfg, v); err != nil {
				return Config{}, fmt.Errorf("config %s: %w", path, err)
			}
		default:
			if !slices.Contains(allowedFlags, k) {
				return Config{}, fmt.Errorf("config %s: unknown or unsupported key %q", path, k)
			}
			switch v.(type) {
			case string, int64, float64, bool:
				cfg.Flags[k] = fmt.Sprint(v)
			default:
				return Config{}, fmt.Errorf("config %s: %s must be a string, number or boolean, got %v", path, k, v)
			}
		}
	}
	return cfg, nil
}

func loadRandom(cfg *Config, v any) error {
	tbl, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("random must be a table")
	}
	for k, bv := range tbl {
		if k != "backgrounds" {
			return fmt.Errorf("unknown key %q", "random."+k)
		}
		list, ok := bv.([]any)
		if !ok {
			return fmt.Errorf("random.backgrounds must be a list of strings, got %v", bv)
		}
		if len(list) == 0 {
			return fmt.Errorf("random.backgrounds: no backgrounds to pick from")
		}
		pool := make([]string, 0, len(list))
		for _, e := range list {
			s, ok := e.(string)
			if !ok {
				return fmt.Errorf("random.backgrounds must be a list of strings, got %v", e)
			}
			pool = append(pool, s)
		}
		cfg.RandomPool = pool
	}
	return nil
}
