package spec

import (
	"strconv"
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
			return name
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

func (f flag) Default() string {
	docFlag, ok := f.Flag.(cli.DocGenerationFlag)
	if !ok || !docFlag.TakesValue() {
		return ""
	}

	text := docFlag.GetDefaultText()
	if text == "" {
		return ""
	}

	// StringFlag, PathFlag, and StringSliceFlag wrap their defaults in
	// display quoting (%q / strconv.Quote) via GetDefaultText. Unwrap so
	// pflag's Value.Set() receives the raw value.
	switch f.Flag.(type) {
	case *cli.StringFlag, *cli.PathFlag:
		if unquoted, err := strconv.Unquote(text); err == nil {
			return unquoted
		}
	case *cli.StringSliceFlag:
		parts := strings.Split(text, ", ")
		unquoted := make([]string, 0, len(parts))
		for _, p := range parts {
			if u, err := strconv.Unquote(p); err == nil {
				unquoted = append(unquoted, u)
			} else {
				unquoted = append(unquoted, p)
			}
		}
		return strings.Join(unquoted, ", ")
	}
	return text
}

func (f flag) Usage() string {
	if docFlag, ok := f.Flag.(cli.DocGenerationFlag); ok {
		return strings.SplitN(docFlag.GetUsage(), "\n", 2)[0]
	}
	return ""
}
