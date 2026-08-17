// Package tui is mori's full-screen interface: the thing you get when you
// type `mori` with nothing after it.
package tui

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/facts"
	"github.com/rmpato/mori/internal/search"
	"github.com/rmpato/mori/internal/store"
	"github.com/rmpato/mori/internal/ui"
)

type mode int

const (
	modeRead mode = iota
	modeWrite
	modeGoto
	modeMood
	modeCalendar
	modeSearch
	modeTags
	modeContext
)

const (
	// mori keeps a narrow column even on a wide terminal. A journal is a
	// reading medium, and the whitespace is the point.
	maxContentWidth = 76
	minContentWidth = 24

	// How long mori waits after the last keystroke before saving. Losing
	// writing is the one unforgivable failure, and a page is a small file.
	autosaveDelay = 750 * time.Millisecond

	// Sitting back down after this long stamps a new section on the page.
	sittingGap = 2 * time.Hour

	// Coming back after this long is worth a hello. Never a number, never a
	// reproach — just an acknowledgement that time passed.
	awayGap = 14 * 24 * time.Hour

	statusLife = 3 * time.Second
)

type autosaveMsg struct{ seq int }
type statusExpiredMsg struct{ seq int }
type editorDoneMsg struct{ err error }

// Model is the Bubble Tea model for mori.
type Model struct {
	store *store.Store
	keys  KeyMap
	theme ui.Theme
	south bool

	date entry.Date // the day on screen
	page entry.Entry

	vp    viewport.Model
	ta    textarea.Model
	input textinput.Model
	help  help.Model

	mode mode

	// The month view: which month is drawn, which days in it have pages, and
	// where the cursor is sitting.
	month   entry.Date
	cursor  entry.Date
	written map[entry.Date]bool

	// Searching and tags: the answers, and which one is selected. findSeq
	// lets a result recognise that a later keystroke has superseded it.
	results  []search.Match
	tags     []search.TagCount
	selected int
	findSeq  int

	// source is what else knows about a day — tuki, when it's installed and
	// you haven't turned it off. nil means mori never mentions it.
	source facts.Source
	snap   facts.Snapshot

	// saveSeq lets a pending autosave recognise that it has been superseded.
	saveSeq int

	status    string
	statusSeq int
	greeting  string

	err error // the first thing that went wrong badly enough to mention

	width, height int
	ready         bool

	// Rendered chrome, rebuilt by refresh so View stays a pure assembly step.
	header, footer string

	// now is a field so tests can pin the clock.
	now func() time.Time
}

// New builds the model on a given day. src is what else knows about a day,
// and may be nil — mori works the same without it, and says nothing about it
// either way.
func New(s *store.Store, d entry.Date, src facts.Source) *Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.SetVirtualCursor(true)

	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 40
	in.Placeholder = "yesterday · fri · -3d · 2026-08-17"
	in.SetVirtualCursor(true)

	m := &Model{
		store:  s,
		source: src,
		keys:   DefaultKeys(),
		south:  ui.SouthernHemisphere(),
		date:   d,
		vp:     viewport.New(),
		ta:     ta,
		input:  in,
		help:   help.New(),
		now:    time.Now,
	}
	m.applyTheme(ui.PrefersDark())
	m.load(d)
	m.greet()
	return m
}

// applyTheme rebuilds every style-carrying component for a light or dark
// terminal. It runs once at startup with a guess, and again for real once the
// terminal says what colour it is.
func (m *Model) applyTheme(isDark bool) {
	m.theme = ui.New(isDark)

	m.help.Styles = help.DefaultStyles(isDark)
	m.help.Styles.ShortKey = m.help.Styles.ShortKey.Foreground(m.theme.Muted)
	m.help.Styles.ShortDesc = m.help.Styles.ShortDesc.Foreground(m.theme.Faint)
	m.help.Styles.ShortSeparator = m.help.Styles.ShortSeparator.Foreground(m.theme.Faint)
	m.help.Styles.FullKey = m.help.Styles.FullKey.Foreground(m.theme.Muted)
	m.help.Styles.FullDesc = m.help.Styles.FullDesc.Foreground(m.theme.Faint)
	m.help.Styles.FullSeparator = m.help.Styles.FullSeparator.Foreground(m.theme.Faint)

	ts := textarea.DefaultStyles(isDark)
	ts.Focused.Base = ts.Focused.Base.Foreground(m.theme.Text)
	ts.Focused.Text = ts.Focused.Text.Foreground(m.theme.Text)
	ts.Focused.CursorLine = ts.Focused.CursorLine.Background(nil)
	ts.Focused.Placeholder = ts.Focused.Placeholder.Foreground(m.theme.Faint)
	ts.Focused.EndOfBuffer = ts.Focused.EndOfBuffer.Foreground(m.theme.Faint)
	ts.Cursor.Color = m.theme.Brand
	m.ta.SetStyles(ts)

	is := textinput.DefaultStyles(isDark)
	is.Focused.Text = is.Focused.Text.Foreground(m.theme.Text)
	is.Focused.Placeholder = is.Focused.Placeholder.Foreground(m.theme.Faint)
	is.Cursor.Color = m.theme.Brand
	m.input.SetStyles(is)
}

// load reads a day into the model. A day that can't be read is reported
// rather than silently shown as blank — a blank page is what an unwritten day
// looks like, and mori must never confuse the two.
func (m *Model) load(d entry.Date) {
	page, err := m.store.Get(d)
	if err != nil {
		m.err = err
		return
	}
	m.date = d
	m.page = page
	m.vp.SetYOffset(0)
}

// greet says hello when you've been away a while. It says it once, without a
// number in it, and there is deliberately no line for the opposite case:
// mori never mentions that you haven't written.
func (m *Model) greet() {
	if !m.page.IsEmpty() {
		return
	}
	dates, err := m.store.Dates(entry.Date{}, m.date.Add(-1))
	if err != nil || len(dates) == 0 {
		return
	}
	last := dates[len(dates)-1]
	if m.date.Since(last) >= int(awayGap.Hours()/24) {
		m.greeting = "welcome back."
	}
}

// Init asks the terminal what colour it is, then gets out of the way.
func (m *Model) Init() tea.Cmd {
	return tea.RequestBackgroundColor
}

// Update handles one message.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.applyTheme(msg.IsDark())

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true

	case autosaveMsg:
		// Only the newest pending save is still meaningful; the others were
		// superseded by later keystrokes.
		if msg.seq == m.saveSeq && m.mode == modeWrite {
			m.commit()
		}

	case statusExpiredMsg:
		if msg.seq == m.statusSeq {
			m.status = ""
		}

	case searchDoneMsg:
		// A result from a keystroke that has since been overtaken is stale.
		if msg.seq != m.findSeq {
			break
		}
		if msg.err != nil {
			m.results = nil
			m.setStatus(msg.err.Error())
			cmds = append(cmds, m.expireStatus())
			break
		}
		m.status = ""
		m.results = msg.matches
		m.move(0, len(m.results))

	case tagsDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			break
		}
		m.tags = msg.counts
		m.move(0, len(m.tags))

	case contextDoneMsg:
		if msg.date != m.date {
			break // you moved on while tuki was being read
		}
		if msg.err != nil {
			m.setStatus(msg.err.Error())
			cmds = append(cmds, m.expireStatus())
			break
		}
		m.snap = msg.snap

	case editorDoneMsg:
		if msg.err != nil {
			m.setStatus(msg.err.Error())
			cmds = append(cmds, m.expireStatus())
		}
		// Whatever your editor did to the file is the truth now.
		m.load(m.date)

	case tea.KeyPressMsg:
		cmd, quit := m.handleKey(msg)
		if quit {
			m.commit()
			return m, tea.Quit
		}
		cmds = append(cmds, cmd)

	case tea.MouseWheelMsg:
		if m.mode == modeRead {
			vp, cmd := m.vp.Update(msg)
			m.vp = vp
			cmds = append(cmds, cmd)
		}
	}

	m.refresh()
	return m, tea.Batch(cmds...)
}

// handleKey routes a keypress to whichever mode is active, and reports
// whether mori should stop.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	// Ctrl+C always works, everywhere, whatever is focused — and it saves on
	// the way out like every other exit.
	if msg.String() == "ctrl+c" {
		return nil, true
	}

	switch m.mode {
	case modeWrite:
		return m.handleWriteKey(msg), false
	case modeGoto, modeMood:
		return m.handlePromptKey(msg), false
	case modeSearch:
		return m.handleSearchKey(msg), false
	case modeTags:
		return m.handleTagsKey(msg), false
	case modeContext:
		return m.handleContextKey(msg), false
	case modeCalendar:
		if m.handleCalendarKey(msg) {
			return nil, false
		}
		// Anything the calendar doesn't claim — q, ? — falls through to the
		// keys that work everywhere.
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return nil, true

	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll

	case key.Matches(msg, m.keys.PrevDay):
		m.goTo(m.date.Add(-1))
	case key.Matches(msg, m.keys.NextDay):
		m.goTo(m.date.Add(1))

	case key.Matches(msg, m.keys.Today):
		m.goTo(entry.DateOf(m.now()))

	case key.Matches(msg, m.keys.Goto):
		return m.openPrompt(modeGoto, ""), false

	case key.Matches(msg, m.keys.Mood):
		return m.openPrompt(modeMood, m.page.Mood), false

	case key.Matches(msg, m.keys.Calendar):
		m.openCalendar()

	case key.Matches(msg, m.keys.Search):
		return m.openSearch(m.input.Value()), false

	case key.Matches(msg, m.keys.Tags):
		return m.openTags(), false

	case key.Matches(msg, m.keys.Editor):
		return m.openEditor(), false

	case key.Matches(msg, m.keys.Context):
		return m.openContext(), false

	case key.Matches(msg, m.keys.Write):
		return m.beginWrite(false), false
	case key.Matches(msg, m.keys.Section):
		return m.beginWrite(true), false

	case key.Matches(msg, m.keys.Up):
		m.vp.ScrollUp(1)
	case key.Matches(msg, m.keys.Down):
		m.vp.ScrollDown(1)
	case key.Matches(msg, m.keys.Top):
		m.vp.GotoTop()
	case key.Matches(msg, m.keys.End):
		m.vp.GotoBottom()
	}

	return nil, false
}

// handleWriteKey drives the page while you're writing in it. Everything that
// isn't esc or ctrl+s belongs to the text.
func (m *Model) handleWriteKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Done):
		m.endWrite()
		return nil

	case key.Matches(msg, m.keys.Save):
		m.commit()
		m.setStatus("saved")
		return m.expireStatus()
	}

	ta, cmd := m.ta.Update(msg)
	m.ta = ta
	return tea.Batch(cmd, m.scheduleSave())
}

// openPrompt puts the cursor in the one-line prompt at the bottom.
func (m *Model) openPrompt(to mode, value string) tea.Cmd {
	m.mode = to
	m.input.SetValue(value)
	m.input.CursorEnd()
	return m.input.Focus()
}

// handlePromptKey drives the one-line prompt, which asks for a date or a
// mood depending on how it was opened.
func (m *Model) handlePromptKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = modeRead
		m.input.Blur()
		return nil

	case key.Matches(msg, m.keys.Submit):
		if m.mode == modeMood {
			m.setMood(entry.NormalizeMood(m.input.Value()))
			m.mode = modeRead
			m.input.Blur()
			return nil
		}

		d, err := entry.ParseDate(m.input.Value(), m.now())
		if err != nil {
			m.setStatus(err.Error())
			return m.expireStatus()
		}
		m.mode = modeRead
		m.input.Blur()
		m.goTo(d)
		return nil
	}

	in, cmd := m.input.Update(msg)
	m.input = in
	return cmd
}

// setMood records a word about the day, or clears it. There is no scale, no
// score, and mori never says anything back about it.
func (m *Model) setMood(mood string) {
	if mood == m.page.Mood {
		return
	}
	m.page.Mood = mood
	m.commit()
}

// openEditor hands the day to $EDITOR and takes it back afterwards.
//
// The page is saved first, so your editor opens what you have written rather
// than what mori last got round to writing down.
func (m *Model) openEditor() tea.Cmd {
	m.commit()

	path, err := m.store.Prepare(m.date)
	if err != nil {
		m.setStatus(err.Error())
		return m.expireStatus()
	}

	editor, args := editorCommand()
	if editor == "" {
		m.setStatus("no $EDITOR set")
		return m.expireStatus()
	}

	c := exec.Command(editor, append(args, path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg { return editorDoneMsg{err: err} })
}

// editorCommand resolves which editor to use, respecting the usual
// convention that VISUAL wins over EDITOR for a full-screen one.
func editorCommand() (string, []string) {
	for _, v := range []string{os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if fields := strings.Fields(v); len(fields) > 0 {
			return fields[0], fields[1:]
		}
	}
	if path, err := exec.LookPath("vi"); err == nil {
		return path, nil
	}
	return "", nil
}

// goTo moves to another day, saving whatever is on the current one first.
func (m *Model) goTo(d entry.Date) {
	if d == m.date {
		return
	}
	m.commit()
	m.load(d)
}

// beginWrite puts the cursor in the page.
//
// Coming back to a day you already wrote in — either after a couple of hours
// or because you asked for a break with 'n' — stamps a section heading and
// starts underneath it. A day written in one sitting never gets a timestamp
// at all.
func (m *Model) beginWrite(force bool) tea.Cmd {
	body := m.page.Body
	if !m.page.IsEmpty() && (force || m.satDownAgain()) {
		body = strings.TrimRight(body, " \t\n") + "\n\n" + entry.SectionHeading(m.now()) + "\n\n"
	} else if body != "" {
		body = strings.TrimRight(body, " \t\n") + "\n\n"
	}

	m.mode = modeWrite
	m.ta.SetValue(body)
	m.ta.MoveToEnd()
	return m.ta.Focus()
}

// satDownAgain reports whether enough time has passed since the page was last
// touched that coming back to it counts as a new sitting.
func (m *Model) satDownAgain() bool {
	at, ok, err := m.store.LastWritten(m.date)
	if err != nil || !ok {
		return false
	}
	return m.now().Sub(at) >= sittingGap
}

// endWrite saves and hands the page back to the reader.
func (m *Model) endWrite() {
	m.commit()
	m.mode = modeRead
	m.ta.Blur()
	m.vp.GotoBottom()
}

// commit pulls whatever is in the editor back into the page and writes it, if
// anything actually changed. It is called on every exit from writing, on
// every day change, on quit, and on the autosave tick — so there is no path
// out of the editor that loses writing.
func (m *Model) commit() {
	if m.mode == modeWrite {
		body := entry.TrimTrailingEmptySection(m.ta.Value())
		m.page.Body = strings.TrimRight(body, " \t\n")
	}
	if m.err != nil {
		return // don't write over a day mori couldn't read in the first place
	}

	saved, err := m.store.Get(m.date)
	if err != nil {
		m.err = err
		return
	}
	if saved.Body == m.page.Body && saved.Mood == m.page.Mood {
		return
	}
	if err := m.store.Put(m.page); err != nil {
		m.err = err
	}
}

// scheduleSave restarts the autosave timer. Every keystroke pushes it out, so
// mori writes when you pause rather than while you're mid-sentence.
func (m *Model) scheduleSave() tea.Cmd {
	m.saveSeq++
	seq := m.saveSeq
	return tea.Tick(autosaveDelay, func(time.Time) tea.Msg { return autosaveMsg{seq: seq} })
}

func (m *Model) setStatus(s string) {
	m.status = s
	m.statusSeq++
}

func (m *Model) expireStatus() tea.Cmd {
	seq := m.statusSeq
	return tea.Tick(statusLife, func(time.Time) tea.Msg { return statusExpiredMsg{seq: seq} })
}

// Err is whatever went wrong badly enough that mori should say so on the way
// out.
func (m *Model) Err() error { return m.err }
