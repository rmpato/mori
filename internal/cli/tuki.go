package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rmpato/mori/internal/config"
	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/facts"
)

// runFromTuki prints the start of a page, built from what tuki knows about
// the day: headings, and the facts under them, and space to write in.
//
// It stops there on purpose. tuki holds what you did; mori holds what it was
// like; and mori will not invent the second from the first.
func runFromTuki(e *env, d entry.Date, write bool) error {
	src, ok := e.tuki()
	if !ok {
		return errors.New("mori can't find tuki — is it installed?")
	}

	snap, err := src.Day(d)
	if err != nil {
		return err
	}
	template := facts.Template(d, snap)

	if !write {
		fmt.Fprint(e.out, template)
		return nil
	}

	page, err := e.store.Get(d)
	if err != nil {
		return err
	}
	// Never over the top of writing. A starting point is only a starting
	// point when there is nothing there yet.
	if !page.IsEmpty() {
		return fmt.Errorf("%s already has writing on it", d.Human())
	}

	page.Body = template
	if err := e.store.Put(page); err != nil {
		return err
	}
	if e.tty {
		e.println(e.theme.Hint.Render("  " + d.Human() + " is ready for you"))
	}
	return nil
}

func newConfigCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "config",
		Short: "what mori is set up to do",
		Long: `Show how mori is configured.

There is deliberately very little to configure. The only real setting is
whether mori reads your tuki task list for context while you write — and by
default it does that if tuki is installed, and never mentions it if not.

Settings live in a small JSON file:

  {
    "tuki": { "enabled": false }
  }

mori only ever reads from tuki. Writing to tuki is not a setting.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			e, err := newEnv(cmd)
			if err != nil {
				return err
			}
			path, err := config.Path()
			if err != nil {
				return err
			}

			present := facts.Available(e.tukiFile)
			enabled := e.cfg.TukiEnabled(present)

			if asJSON {
				return writeJSON(e.out, struct {
					Config  string `json:"config"`
					Journal string `json:"journal"`
					Tuki    struct {
						Enabled   bool   `json:"enabled"`
						Installed bool   `json:"installed"`
						File      string `json:"file"`
						Write     bool   `json:"write"`
					} `json:"tuki"`
				}{
					Config:  path,
					Journal: e.store.Root(),
					Tuki: struct {
						Enabled   bool   `json:"enabled"`
						Installed bool   `json:"installed"`
						File      string `json:"file"`
						Write     bool   `json:"write"`
					}{enabled, present, e.tukiFile, false},
				})
			}

			e.println()
			e.println("  " + e.theme.Section.Render("journal"))
			e.println("  " + e.theme.Hint.Render(e.store.Root()))
			e.println()
			e.println("  " + e.theme.Section.Render("config"))
			e.println("  " + e.theme.Hint.Render(path))
			e.println()
			e.println("  " + e.theme.Section.Render("tuki"))
			e.println("  " + e.mark(present) + " installed  " + e.theme.Hint.Render(e.tukiFile))
			e.println("  " + e.mark(enabled) + " read tasks")
			e.println("  " + e.mark(enabled) + " read tags")
			e.println("  " + e.mark(false) + " write to tuki  " +
				e.theme.Hint.Render("mori reads from tuki; mori doesn't control tuki"))
			e.println()
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the configuration as JSON")
	return cmd
}

func (e *env) mark(on bool) string {
	if on {
		return e.theme.Tag.Render("✓")
	}
	return e.theme.Hint.Render("✗")
}
