package cli

import (
	"github.com/spf13/cobra"
)

// showOptions are the ways to ask for a page without mori's decoration.
type showOptions struct {
	plain    bool
	json     bool
	fromTuki bool
	write    bool
}

func (o *showOptions) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.plain, "plain", false, "print the raw Markdown, unstyled")
	cmd.Flags().BoolVar(&o.json, "json", false, "print the page as JSON")
}

// bindTuki adds the flags that reach across to tuki. They live on `today`,
// because a scaffold made of today's tasks is only ever wanted for today or
// thereabouts.
func (o *showOptions) bindTuki(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.fromTuki, "from-tuki", false, "print the start of a page, built from what tuki did today")
	cmd.Flags().BoolVar(&o.write, "write", false, "with --from-tuki, put it in the page (only if the page is empty)")
}

func newTodayCmd() *cobra.Command {
	var opts showOptions
	cmd := &cobra.Command{
		Use:   "today",
		Short: "print today's page",
		Long: `Print today's page.

With --from-tuki, print the start of one instead: the headings, and today's
finished and unfinished tasks under them, and space to write in.

  mori today --from-tuki
  mori today --from-tuki --write

That is a scaffold, not a journal entry. tuki holds what you did; mori holds
what it was like; and mori will not write the second from the first.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runShow(cmd, nil, opts)
		},
	}
	opts.bind(cmd)
	opts.bindTuki(cmd)
	return cmd
}

func newShowCmd() *cobra.Command {
	var opts showOptions
	cmd := &cobra.Command{
		Use:   "show [date]",
		Short: "print the page for a day",
		Long: `Print the page for a day.

The date can be written any way you'd say it:

  mori show yesterday
  mori show fri
  mori show -3d
  mori show 2026-08-17
  mori show "17 aug"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, args, opts)
		},
	}
	opts.bind(cmd)
	return cmd
}

func runShow(cmd *cobra.Command, args []string, opts showOptions) error {
	e, err := newEnv(cmd)
	if err != nil {
		return err
	}
	d, err := e.date(args)
	if err != nil {
		return err
	}
	if opts.fromTuki {
		return runFromTuki(e, d, opts.write)
	}
	en, err := e.store.Get(d)
	if err != nil {
		return err
	}

	if opts.json {
		return writeJSON(e.out, toJSON(en))
	}
	// Piping a page somewhere should give you the page, not a drawing of it.
	e.writeEntry(en, opts.plain || !e.tty)
	return nil
}
