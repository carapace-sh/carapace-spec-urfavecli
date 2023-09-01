package spec

import (
	"strings"

	"github.com/urfave/cli/v2"
)

type flag struct {
	cli.Flag
}

func (f flag) Name() string {
	return f.Names()[0]
}

func (f flag) Shorthand() string {
	for _, name := range f.Names() {
		if len(name) == 1 {
			return "-" + name
		}
	}
	return ""
}

func (f flag) TakesValue() bool {
	if docFlag, ok := f.Flag.(cli.DocGenerationFlag); ok {
		return docFlag.TakesValue()
	}
	return false
}

func (f flag) TakesFile() bool {
	switch flag := f.Flag.(type) {
	case *cli.GenericFlag:
		if flag.TakesFile {
			return true
		}
	case *cli.StringFlag:
		if flag.TakesFile {
			return true
		}
	case *cli.StringSliceFlag:
		if flag.TakesFile {
			return true
		}
	case *cli.PathFlag:
		if flag.TakesFile {
			return true
		}
	}
	return false
}

func (f flag) Usage() string {
	if docFlag, ok := f.Flag.(cli.DocGenerationFlag); ok {
		return strings.SplitN(docFlag.GetUsage(), "\n", 2)[0]
	}
	return ""
}
