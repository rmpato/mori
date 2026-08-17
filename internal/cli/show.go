package cli

import (
	"github.com/spf13/cobra"
)

// showOptions are the two ways to ask for a page without mori's decoration.
type showOptions struct {
	plain bool
	json  bool
}

func (o *showOptions) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.plain, "plain", false, "print the raw Markdown, unstyled")
	cmd.Flags().BoolVar(&o.json, "json", false, "print the page as JSON")
}

func newTodayCmd() *cobra.Command {
	var opts showOptions
	cmd := &cobra.Command{
		Use:   "today",
		Short: "print today's page",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runShow(cmd, nil, opts)
		},
	}
	opts.bind(cmd)
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
