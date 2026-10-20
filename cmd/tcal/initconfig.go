package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/v4run/tcal/internal/tui"
)

// configWritten is returned by configure after --init-config wrote a file.
type configWritten struct{ path string }

func (c configWritten) Error() string { return "wrote config " + c.path }

// writeConfig creates path holding configTemplate(flags). It never replaces
// an existing file.
func writeConfig(path string, flags *flag.FlagSet) error {
	if path == "" {
		return errors.New("--init-config: no config location (set --config, XDG_CONFIG_HOME or HOME)")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("--init-config: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("config %s already exists; delete it or pass --config to write elsewhere", path)
	}
	if err != nil {
		return fmt.Errorf("--init-config: %w", err)
	}
	if _, err := f.WriteString(configTemplate(flags)); err != nil {
		f.Close()
		return fmt.Errorf("--init-config: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("--init-config: %w", err)
	}
	return configWritten{path}
}

// configTemplate is a config file setting every allowed flag to its built-in
// default, with the personal and random-pool settings commented out. Loading
// it behaves exactly like having no file.
func configTemplate(flags *flag.FlagSet) string {
	var b strings.Builder
	b.WriteString("# tcal config. Every key is optional; command-line flags win over it.\n")
	b.WriteString("# The values below are the built-in defaults.\n\n")
	flags.VisitAll(func(f *flag.Flag) {
		if slices.Contains(configForbidden, f.Name) {
			return
		}
		fmt.Fprintf(&b, "# %s\n%s = %s\n\n", f.Usage, f.Name, tomlValue(f))
	})
	b.WriteString("# Your birthday shows a cake in the seasons background: MM-DD (YYYY-MM-DD also accepted).\n")
	b.WriteString("# birthday = \"05-17\"\n\n")
	b.WriteString("# Backgrounds --bg random may pick. Omit to allow all of them.\n")
	b.WriteString("# [random]\n")
	names := make([]string, 0, len(tui.Backgrounds()))
	for _, n := range tui.Backgrounds() {
		names = append(names, strconv.Quote(n))
	}
	fmt.Fprintf(&b, "# backgrounds = [%s]\n", strings.Join(names, ", "))
	return b.String()
}

// tomlValue renders a flag's default as a TOML value.
func tomlValue(f *flag.Flag) string {
	if g, ok := f.Value.(flag.Getter); ok {
		switch v := g.Get().(type) {
		case bool:
			return strconv.FormatBool(v)
		case int:
			return strconv.Itoa(v)
		}
	}
	return strconv.Quote(f.DefValue)
}
