package cli

import (
	"context"
	"os"
	"regexp"
	"runtime/debug"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/tui"
)

// Version and Commit are stamped at build time with -ldflags. When they
// aren't, mori asks the Go build info, which `go install` fills in.
var (
	Version = ""
	Commit  = ""
)

func version() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func commit() string {
	if Commit != "" {
		return Commit
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				return s.Value
			}
		}
	}
	return ""
}

// NewRoot builds the whole command tree.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "mori",
		Short: "a quiet place to remember your days",
		Long: `mori keeps a journal in your terminal.

Running mori with no arguments opens today's page, ready to write in.
Everything else is here so mori works in scripts and pipes too.

  mori today
  mori show yesterday
  mori search photography
  mori tags

Pages are plain Markdown files on your machine, one per day. Nothing is sent
anywhere.`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}
			// Piping `mori` somewhere should give you today's page, not an
			// interface nobody can see.
			if !e.tty {
				return runShow(cmd, nil, showOptions{})
			}
			src, _ := e.tuki()
			return tui.Run(e.store, entry.DateOf(e.now), src, e.out)
		},
	}

	root.PersistentFlags().String("dir", "", "use a specific journal directory instead of the default")

	root.AddCommand(
		newTodayCmd(),
		newShowCmd(),
		newListCmd(),
		newSearchCmd(),
		newTagsCmd(),
		newPathCmd(),
		newConfigCmd(),
		newUpdateCmd(),
	)
	return root
}

// dashedOffsetRe matches a backwards offset written with a leading dash.
var dashedOffsetRe = regexp.MustCompile(`^-(\d+)([dwmy])$`)

// normalizeArgs rewrites "-3d" as "3d" before the flag parser gets to it.
//
// The two mean exactly the same thing to mori — offsets look backwards unless
// you write a "+" — but a leading dash makes pflag think it has found a
// shorthand flag and refuse. Rewriting loses nothing and means `mori show -3d`
// does what it plainly looks like.
func normalizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if dashedOffsetRe.MatchString(a) {
			out[i] = a[1:]
			continue
		}
		out[i] = a
	}
	return out
}

// Execute runs mori, wrapped in fang for the styled help, errors, manpage,
// and shell completions.
func Execute(ctx context.Context) error {
	opts := []fang.Option{fang.WithVersion(version())}
	if c := commit(); c != "" {
		opts = append(opts, fang.WithCommit(c))
	}
	root := NewRoot()
	root.SetArgs(normalizeArgs(os.Args[1:]))
	return fang.Execute(ctx, root, opts...)
}
