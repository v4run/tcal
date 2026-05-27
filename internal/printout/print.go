// Package printout renders one frame to an io.Writer and exits.
package printout

import (
	"errors"
	"io"
	"syscall"

	"github.com/v4run/tcal/internal/render"
)

// Run renders state once via render.Frame and writes the result + trailing
// newline to w. EPIPE (broken pipe) is treated as success so `tcal --print
// | head` is well-behaved.
func Run(state render.State, opts render.Options, w io.Writer) error {
	state.Height = 0 // print mode never vertically centers
	out := render.Frame(state, opts) + "\n"
	_, err := io.WriteString(w, out)
	if err != nil {
		if errors.Is(err, syscall.EPIPE) {
			return nil
		}
		return err
	}
	return nil
}
