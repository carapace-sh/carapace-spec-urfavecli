package spec

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestRegisterInjectsHiddenCommand(t *testing.T) {
	app := &cli.Command{
		Name:  "testapp",
		Usage: "test app",
	}
	Register(app)

	found := false
	for _, cmd := range app.Commands {
		if cmd.Name == "_carapace" && cmd.Hidden {
			found = true
			if len(cmd.Commands) != 1 || cmd.Commands[0].Name != "spec" {
				t.Errorf("expected single 'spec' subcommand, got %v", cmd.Commands)
			}
		}
	}
	if !found {
		t.Errorf("Register did not inject hidden _carapace command")
	}

	// verify _carapace doesn't appear in spec output (it's hidden)
	cmd := Command(app)
	for _, sub := range cmd.Commands {
		if sub.Name == "_carapace" {
			t.Errorf("_carapace should be filtered from spec output (hidden)")
		}
	}
}

func TestRegisterSpecAction(t *testing.T) {
	app := &cli.Command{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Usage: "output path",
				Value: "/tmp/out.txt",
			},
		},
		Writer:    &bytes.Buffer{},
		ErrWriter: &bytes.Buffer{},
	}
	Register(app)

	// capture stdout (fmt.Println writes to os.Stdout, not app.Writer)
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	err = app.Run(context.Background(), []string{"testapp", "_carapace", "spec"})

	w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatalf("app.Run failed: %v", err)
	}

	out, _ := io.ReadAll(r)
	output := string(out)
	if !strings.Contains(output, "name: testapp") {
		t.Errorf("spec output missing app name, got:\n%s", output)
	}
	if !strings.Contains(output, "default: /tmp/out.txt") {
		t.Errorf("spec output missing flag default, got:\n%s", output)
	}
}
