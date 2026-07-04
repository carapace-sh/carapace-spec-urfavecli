package spec

import (
	"strings"
	"testing"

	"github.com/carapace-sh/carapace-spec/pkg/command"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

// findFlag returns the FlagSet entry whose Longhand matches the given bare name.
func findFlag(t *testing.T, cmd command.Command, longhand string) command.Flag {
	t.Helper()
	for _, f := range cmd.Flags {
		if f.Longhand == longhand {
			return f
		}
	}
	t.Fatalf("flag %q not found (keys: %v)", longhand, flagKeys(cmd.Flags))
	return command.Flag{}
}

func flagKeys(fs command.FlagSet) []string {
	keys := make([]string, 0, len(fs))
	for k := range fs {
		keys = append(keys, k)
	}
	return keys
}

func TestDefaultValues(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Usage: "output path",
				Value: "/tmp/out.txt",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "port number",
				Value: 8080,
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose mode",
			},
			&cli.PathFlag{
				Name:  "config",
				Usage: "config path",
				Value: "/etc/app.yaml",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Usage: "timeout duration",
				Value: 5000000000, // 5s
			},
			&cli.StringFlag{
				Name:  "no-default",
				Usage: "no default set",
			},
		},
	}

	cmd := Command(app)

	tests := []struct {
		longhand string
		want     string
	}{
		{"output", "/tmp/out.txt"},
		{"port", "8080"},
		{"config", "/etc/app.yaml"},
		{"timeout", "5s"},
		{"no-default", ""},
	}
	for _, tt := range tests {
		f := findFlag(t, cmd, tt.longhand)
		if f.Default != tt.want {
			t.Errorf("flag %q default: got %q, want %q", tt.longhand, f.Default, tt.want)
		}
	}

	// bool flags take no value, so Default should be empty
	verbose := findFlag(t, cmd, "verbose")
	if verbose.Default != "" {
		t.Errorf("verbose flag default: got %q, want empty (bool flags skip default)", verbose.Default)
	}
}

func TestStringSliceFlagDefault(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "files",
				Usage: "config files",
				Value: cli.NewStringSlice("/etc/a.conf", "/etc/b.conf"),
			},
		},
	}

	cmd := Command(app)
	f := findFlag(t, cmd, "files")
	want := "/etc/a.conf, /etc/b.conf"
	if f.Default != want {
		t.Errorf("StringSliceFlag default: got %q, want %q", f.Default, want)
	}
}

func TestShorthand(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output path",
			},
			&cli.StringFlag{
				Name:  "verbose",
				Usage: "verbose mode",
			},
		},
	}

	cmd := Command(app)

	output := findFlag(t, cmd, "output")
	if output.Shorthand != "o" {
		t.Errorf("output shorthand: got %q, want %q", output.Shorthand, "o")
	}

	verbose := findFlag(t, cmd, "verbose")
	if verbose.Shorthand != "" {
		t.Errorf("verbose shorthand: got %q, want empty", verbose.Shorthand)
	}
}

func TestFlagKeyFormat(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output, o",
				Usage: "output path",
				Value: "/tmp/out.txt",
			},
		},
	}

	cmd := Command(app)

	// Keys should use the correct --prefix (not ----prefix)
	for k := range cmd.Flags {
		if strings.HasPrefix(k, "----") {
			t.Errorf("flag key %q has double -- prefix (should be single)", k)
		}
	}
}

func TestTakesFile(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "output",
				Usage:     "output path",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "name value",
			},
			&cli.PathFlag{
				Name:      "config",
				Usage:     "config path",
				TakesFile: true,
			},
		},
	}

	cmd := Command(app)

	if _, ok := cmd.Completion.Flag["output"]; !ok {
		t.Errorf("expected file completion for 'output' flag")
	}
	if _, ok := cmd.Completion.Flag["config"]; !ok {
		t.Errorf("expected file completion for 'config' flag")
	}
	if _, ok := cmd.Completion.Flag["name"]; ok {
		t.Errorf("did not expect file completion for 'name' flag")
	}
}

func TestUsageTruncation(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "desc",
				Usage: "first line\nsecond line",
			},
		},
	}

	cmd := Command(app)
	f := findFlag(t, cmd, "desc")
	if f.Description != "first line" {
		t.Errorf("usage truncation: got %q, want %q", f.Description, "first line")
	}
}

func TestHiddenCommandsFiltered(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Commands: []*cli.Command{
			{
				Name:   "visible",
				Usage:  "visible command",
				Hidden: false,
			},
			{
				Name:   "secret",
				Usage:  "hidden command",
				Hidden: true,
			},
		},
	}

	cmd := Command(app)
	if len(cmd.Commands) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(cmd.Commands))
	}
	if cmd.Commands[0].Name != "visible" {
		t.Errorf("expected 'visible' subcommand, got %q", cmd.Commands[0].Name)
	}
}

func TestSubcommandRecursion(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Commands: []*cli.Command{
			{
				Name:  "parent",
				Usage: "parent command",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "parent-flag",
						Usage: "parent flag",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:  "child",
						Usage: "child command",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "child-flag",
								Usage: "child flag",
							},
						},
					},
				},
			},
		},
	}

	cmd := Command(app)
	if len(cmd.Commands) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(cmd.Commands))
	}
	parent := cmd.Commands[0]
	if parent.Name != "parent" {
		t.Errorf("expected 'parent', got %q", parent.Name)
	}
	if len(parent.Commands) != 1 {
		t.Fatalf("expected 1 grandchild, got %d", len(parent.Commands))
	}
	child := parent.Commands[0]
	if child.Name != "child" {
		t.Errorf("expected 'child', got %q", child.Name)
	}
	// verify child has its own flag
	findFlag(t, child, "child-flag")
}

func TestExtendedYAMLNotation(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Usage: "output path",
				Value: "/tmp/out.txt",
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "port number",
				Value: 8080,
			},
		},
	}

	cmd := Command(app)
	m, err := yaml.Marshal(cmd)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	yamlStr := string(m)

	if !strings.Contains(yamlStr, "default: /tmp/out.txt") {
		t.Errorf("expected YAML to contain raw default for output, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "default: \"8080\"") {
		t.Errorf("expected YAML to contain default for port, got:\n%s", yamlStr)
	}
	// verify no double-dash prefix in flag keys
	if strings.Contains(yamlStr, "----") {
		t.Errorf("YAML contains ---- prefix, got:\n%s", yamlStr)
	}
}

func TestAliasesAndGroup(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Commands: []*cli.Command{
			{
				Name:     "build",
				Aliases:  []string{"b", "make"},
				Usage:    "build something",
				Category: "development",
			},
		},
	}

	cmd := Command(app)
	if len(cmd.Commands) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(cmd.Commands))
	}
	sub := cmd.Commands[0]
	if sub.Name != "build" {
		t.Errorf("name: got %q, want %q", sub.Name, "build")
	}
	if len(sub.Aliases) != 2 || sub.Aliases[0] != "b" || sub.Aliases[1] != "make" {
		t.Errorf("aliases: got %v, want [b make]", sub.Aliases)
	}
	if sub.Group != "development" {
		t.Errorf("group: got %q, want %q", sub.Group, "development")
	}
}

func TestTakesFileAllTypes(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:      "format",
				Usage:     "output format",
				TakesFile: true,
			},
			&cli.StringSliceFlag{
				Name:      "configs",
				Usage:     "config files",
				TakesFile: true,
			},
			&cli.IntFlag{
				Name:  "count",
				Usage: "count value",
			},
		},
	}

	cmd := Command(app)
	if _, ok := cmd.Completion.Flag["format"]; !ok {
		t.Errorf("expected file completion for GenericFlag 'format'")
	}
	if _, ok := cmd.Completion.Flag["configs"]; !ok {
		t.Errorf("expected file completion for StringSliceFlag 'configs'")
	}
	if _, ok := cmd.Completion.Flag["count"]; ok {
		t.Errorf("did not expect file completion for IntFlag 'count'")
	}
}

func TestDefaultWithCustomDefaultText(t *testing.T) {
	app := &cli.App{
		Name:  "testapp",
		Usage: "test app",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Usage:       "run mode",
				Value:       "default",
				DefaultText: "custom-default",
			},
		},
	}

	cmd := Command(app)
	f := findFlag(t, cmd, "mode")
	// DefaultText is returned as-is by GetDefaultText (no %q quoting)
	if f.Default != "custom-default" {
		t.Errorf("custom DefaultText: got %q, want %q", f.Default, "custom-default")
	}
}
