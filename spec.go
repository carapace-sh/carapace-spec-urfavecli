package spec

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

func Scrape(app *cli.App) {
	rootCmd := &cli.Command{
		Name:        app.Name,
		Description: app.Usage,
		Flags:       app.Flags,
		Subcommands: app.Commands,
	}

	cmd := command(rootCmd)
	m, err := yaml.Marshal(cmd)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(string(m))
}

func command(c *cli.Command) Command {
	cmd := Command{
		Name:        c.Name,
		Aliases:     c.Aliases,
		Description: c.Usage,
		Group:       c.Category,
		Flags:       make(map[string]string),
		Commands:    make([]Command, 0),
	}
	cmd.Completion.Flag = make(map[string][]string)

	for _, f := range c.Flags {
		flag := flag{f}
		cmd.Flags[flag.Definition()] = flag.Usage()

		if flag.TakesFile() {
			cmd.Completion.Flag[flag.Name()] = []string{"$files"}
		}
	}

	for _, subcmd := range c.Subcommands {
		if !subcmd.Hidden {
			cmd.Commands = append(cmd.Commands, command(subcmd))
		}
	}
	return cmd
}
