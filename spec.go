package spec

import (
	"fmt"

	"github.com/rsteube/carapace-spec/pkg/command"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

func Register(app *cli.App) {
	app.Commands = append(app.Commands, &cli.Command{
		Name:   "_carapace",
		Hidden: true,
		Subcommands: []*cli.Command{
			{
				Name: "spec",
				Action: func(ctx *cli.Context) (err error) {
					var m []byte
					if m, err = yaml.Marshal(Command(app)); err == nil {
						fmt.Println(string(m))
					}
					return
				},
			},
		},
	})
}

func Command(app *cli.App) command.Command {
	return scrape(&cli.Command{
		Name:        app.Name,
		Description: app.Usage,
		Flags:       app.Flags,
		Subcommands: app.Commands,
	})
}
func scrape(c *cli.Command) command.Command {
	cmd := command.Command{
		Name:        c.Name,
		Aliases:     c.Aliases,
		Description: c.Usage,
		Hidden:      c.Hidden,
		Group:       c.Category,
		Flags:       make(map[string]string),
		Commands:    make([]command.Command, 0),
	}
	cmd.Completion.Flag = make(map[string][]string)

	for _, f := range c.Flags {
		flag := flag{f}
		cmd.AddFlag(command.Flag{
			Longhand:  "--" + flag.Name(),
			Shorthand: flag.Shorthand(),
			Usage:     flag.Usage(),
			Value:     flag.TakesValue(),
		})

		if flag.TakesFile() {
			cmd.Completion.Flag[flag.Name()] = []string{"$files"}
		}
	}

	for _, subcmd := range c.Subcommands {
		if !subcmd.Hidden {
			cmd.Commands = append(cmd.Commands, scrape(subcmd))
		}
	}
	return cmd
}
