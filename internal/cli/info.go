package cli

import (
	"github.com/spf13/cobra"
)

func newPathCmd() *cobra.Command {
	var forDate bool
	cmd := &cobra.Command{
		Use:   "path [date]",
		Short: "print where the journal lives",
		Long: `Print where the journal lives.

With no arguments it prints the journal directory. With a date, it prints the
file that day is kept in, whether or not you've written it yet — which is
what you want for piping into an editor:

  $EDITOR "$(mori path yesterday)"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}
			if len(args) == 0 && !forDate {
				e.println(e.store.Root())
				return nil
			}
			d, err := e.date(args)
			if err != nil {
				return err
			}
			e.println(e.store.Path(d))
			return nil
		},
	}
	cmd.Flags().BoolVar(&forDate, "today", false, "print today's file rather than the directory")
	return cmd
}
