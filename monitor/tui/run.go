// Package tui presents a read-only Gobble dashboard in an interactive terminal.
// Only this leaf package depends on Bubble Tea and Lip Gloss.
package tui

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/HahyeonJeon/gobble"
	"github.com/HahyeonJeon/gobble/monitor"
	"github.com/charmbracelet/x/term"
)

// Watch displays a workspace without owning or controlling its execution.
// output is normally stderr, preserving stdout as the CLI protocol channel.
// Quitting this program never releases or cancels a pipeline.
func Watch(ctx context.Context, workspace string, input, output *os.File, opts ...gobble.OccupyOption) error {
	if input == nil || output == nil || !term.IsTerminal(input.Fd()) || !term.IsTerminal(output.Fd()) || os.Getenv("TERM") == "dumb" {
		return fmt.Errorf("watch requires terminal input and stderr; use inspect monitor --workspace DIR for JSON")
	}
	read := func(instance string) (monitor.Snapshot, error) {
		return monitor.Read(workspace, instance, opts...)
	}
	initial, err := read("")
	if err != nil {
		return err
	}
	m, err := newModel(workspace, read, initial, os.Getenv("NO_COLOR") != "")
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, tea.WithInput(input), tea.WithOutput(output), tea.WithContext(ctx), tea.WithFPS(12)).Run()
	if errors.Is(err, tea.ErrInterrupted) || ctx.Err() != nil {
		return nil
	}
	return err
}
