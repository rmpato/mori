package tui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/rmpato/mori/internal/entry"
	"github.com/rmpato/mori/internal/store"
)

var aug17 = entry.Date{Year: 2026, Month: time.August, Day: 17}

// clock is a Monday afternoon, so "today" and the weekday are both pinned.
var clock = time.Date(2026, time.August, 17, 14, 30, 0, 0, time.UTC)

// harness drives a Model the way Bubble Tea would, so the tests exercise the
// real Update/View path rather than the internals.
type harness struct {
	t     *testing.T
	m     *Model
	store *store.Store
}

func newHarness(t *testing.T, seed map[entry.Date]string) *harness {
	t.Helper()

	s := store.New(filepath.Join(t.TempDir(), "journal"))
	for d, body := range seed {
		if err := s.Put(entry.Entry{Date: d, Body: body}); err != nil {
			t.Fatal(err)
		}
	}

	m := New(s, aug17)
	m.now = func() time.Time { return clock }

	h := &harness{t: t, m: m, store: s}
	h.m.Init()
	h.send(tea.WindowSizeMsg{Width: 90, Height: 24})
	return h
}

// How long the harness waits for a command to answer. The work mori does off
// the main loop — searching a temp journal of a few files, counting its tags —
// takes microseconds. The commands that don't answer in this window are the
// timers, at 750ms and 3s, and the tests that care about those send their
// messages by hand.
const cmdWindow = 25 * time.Millisecond

// send delivers a message, runs whatever commands come back, and feeds their
// messages in as Bubble Tea would — so a test exercises the real asynchronous
// path rather than reaching into the model.
func (h *harness) send(msg tea.Msg) {
	h.t.Helper()
	h.deliver(msg, 0)
}

func (h *harness) deliver(msg tea.Msg, depth int) {
	h.t.Helper()
	model, cmd := h.m.Update(msg)
	h.m = model.(*Model)
	h.run(cmd, depth)
}

// run executes a command tree. Batched commands run at the same time, so one
// sleeping timer alongside a search doesn't cost the search its window.
func (h *harness) run(cmd tea.Cmd, depth int) {
	h.t.Helper()
	if cmd == nil || depth > 8 {
		return
	}

	out := make(chan tea.Msg, 1)
	go func() { out <- cmd() }()

	var msg tea.Msg
	select {
	case msg = <-out:
	case <-time.After(cmdWindow):
		return
	}

	if batch, ok := msg.(tea.BatchMsg); ok {
		var wg sync.WaitGroup
		msgs := make(chan tea.Msg, len(batch))
		for _, c := range batch {
			if c == nil {
				continue
			}
			wg.Add(1)
			go func() { defer wg.Done(); msgs <- c() }()
		}
		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(cmdWindow):
		}
		// The channel is buffered for the whole batch and never closed: a
		// command still sleeping when the window ran out has somewhere to put
		// its message, rather than a closed channel to panic on.
		for {
			select {
			case m := <-msgs:
				if m != nil {
					h.deliver(m, depth+1)
				}
			default:
				return
			}
		}
	}
	if msg != nil {
		h.deliver(msg, depth+1)
	}
}

func (h *harness) press(keys ...string) {
	h.t.Helper()
	for _, k := range keys {
		h.send(keyPress(k))
	}
}

func (h *harness) typeText(s string) {
	h.t.Helper()
	for _, r := range s {
		switch r {
		case ' ':
			h.send(tea.KeyPressMsg{Code: ' ', Text: " "})
		case '\n':
			h.send(tea.KeyPressMsg{Code: tea.KeyEnter})
		default:
			h.send(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
	}
}

// screen is the rendered frame with the styling stripped, which is what the
// user actually sees laid out.
func (h *harness) screen() string {
	h.t.Helper()
	return ansi.Strip(h.m.View().Content)
}

// onDisk is the page as it was actually saved.
func (h *harness) onDisk(d entry.Date) string {
	h.t.Helper()
	e, err := h.store.Get(d)
	if err != nil {
		h.t.Fatal(err)
	}
	return e.Body
}

// age backdates a page's file, which is how mori decides you've sat back down.
func (h *harness) age(d entry.Date, by time.Duration) {
	h.t.Helper()
	at := clock.Add(-by)
	if err := os.Chtimes(h.store.Path(d), at, at); err != nil {
		h.t.Fatal(err)
	}
}

func keyPress(name string) tea.KeyPressMsg {
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "space":
		return tea.KeyPressMsg{Code: ' ', Text: " "}
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	default:
		r := []rune(name)[0]
		return tea.KeyPressMsg{Code: r, Text: name}
	}
}

// mori opens on the day you asked for, with the date where you'd look for it.
func TestOpensOnTheDay(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "Today was actually pretty good."})

	screen := h.screen()
	for _, want := range []string{"Monday", "August 17, 2026", "today", "Today was actually pretty good."} {
		if !strings.Contains(screen, want) {
			t.Errorf("the screen is missing %q:\n%s", want, screen)
		}
	}
}

// An empty day is an empty page. No prompt, no invitation, and above all no
// remark about the blankness.
func TestAnEmptyDayIsJustBlank(t *testing.T) {
	h := newHarness(t, nil)

	body := h.screen()
	if i := strings.Index(body, "─"); i >= 0 {
		if j := strings.LastIndex(body, "─"); j > i {
			body = body[i:j]
		}
	}
	if strings.TrimSpace(strings.ReplaceAll(body, "─", "")) != "" {
		t.Errorf("an empty day said something:\n%s", h.screen())
	}
}

func TestWriting(t *testing.T) {
	h := newHarness(t, nil)

	h.press("enter")
	h.typeText("a quiet day")
	h.press("esc")

	if got := h.onDisk(aug17); got != "a quiet day" {
		t.Errorf("on disk = %q", got)
	}
	if !strings.Contains(h.screen(), "a quiet day") {
		t.Errorf("the page isn't on screen:\n%s", h.screen())
	}
	if h.m.mode != modeRead {
		t.Error("esc didn't hand the page back to the reader")
	}
}

func TestWritingContinuesThePage(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "morning"})

	h.press("enter")
	h.typeText("evening")
	h.press("esc")

	if got := h.onDisk(aug17); got != "morning\n\nevening" {
		t.Errorf("on disk = %q, want the new writing under the old", got)
	}
}

// Coming back to a day hours later is a new sitting, and gets a timestamp.
func TestSittingBackDownStampsTheSection(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "morning"})
	h.age(aug17, 5*time.Hour)

	h.press("enter")
	h.typeText("evening")
	h.press("esc")

	want := "morning\n\n## 14:30\n\nevening"
	if got := h.onDisk(aug17); got != want {
		t.Errorf("on disk = %q, want %q", got, want)
	}
}

// Writing again a minute later is the same sitting, and gets nothing.
func TestTheSameSittingIsNotStamped(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "morning"})
	h.age(aug17, time.Minute)

	h.press("enter")
	h.typeText("still morning")
	h.press("esc")

	if got := h.onDisk(aug17); strings.Contains(got, "##") {
		t.Errorf("on disk = %q, want no timestamp", got)
	}
}

func TestNewSectionOnDemand(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "morning"})

	h.press("n")
	h.typeText("a deliberate break")
	h.press("esc")

	want := "morning\n\n## 14:30\n\na deliberate break"
	if got := h.onDisk(aug17); got != want {
		t.Errorf("on disk = %q, want %q", got, want)
	}
}

// A sitting that didn't happen shouldn't leave a mark.
func TestAStampWithNothingUnderItIsDropped(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "morning"})

	h.press("n")
	h.press("esc")

	if got := h.onDisk(aug17); got != "morning" {
		t.Errorf("on disk = %q, want the page untouched", got)
	}
}

func TestDayNavigation(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{
		aug17:          "monday",
		aug17.Add(-1):  "sunday",
		aug17.Add(-30): "long ago",
	})

	h.press("h")
	if h.m.date != aug17.Add(-1) {
		t.Fatalf("h went to %v", h.m.date)
	}
	if !strings.Contains(h.screen(), "sunday") {
		t.Errorf("the previous day isn't on screen:\n%s", h.screen())
	}

	h.press("l")
	if h.m.date != aug17 {
		t.Fatalf("l went to %v", h.m.date)
	}

	// Left and right work the same as h and l.
	h.press("left", "left")
	if want := aug17.Add(-2); h.m.date != want {
		t.Fatalf("arrows went to %v, want %v", h.m.date, want)
	}

	h.press("t")
	if h.m.date != aug17 {
		t.Errorf("t went to %v, want today", h.m.date)
	}
}

// Walking off the end of what you've written is fine: those days are blank,
// not missing.
func TestNavigatingIntoDaysYouNeverWrote(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "monday"})

	h.press("h", "h", "h")
	if h.m.Err() != nil {
		t.Fatalf("walking backwards failed: %v", h.m.Err())
	}
	if !h.m.page.IsEmpty() {
		t.Errorf("a day never written came back as %q", h.m.page.Body)
	}
}

// While you're writing, the keyboard belongs to the writing. The letters
// that navigate in the reader are just letters here.
func TestNavigationKeysAreTextWhileWriting(t *testing.T) {
	h := newHarness(t, nil)

	h.press("enter")
	h.typeText("qhlntg")

	if h.m.mode != modeWrite {
		t.Fatal("a letter took the cursor out of the page")
	}
	if h.m.date != aug17 {
		t.Errorf("a letter moved the day to %v", h.m.date)
	}
	h.press("esc")
	if got := h.onDisk(aug17); got != "qhlntg" {
		t.Errorf("on disk = %q", got)
	}
}

// There must be no way out of the editor that loses writing. esc is one; the
// interrupt that works everywhere is the other.
func TestWritingSurvivesQuitting(t *testing.T) {
	h := newHarness(t, nil)

	h.press("enter")
	h.typeText("words on the way out")
	h.press("ctrl+c")

	if got := h.onDisk(aug17); got != "words on the way out" {
		t.Errorf("on disk = %q", got)
	}
}

// And nothing is lost between the editor and another day.
func TestWritingSurvivesLeavingTheDay(t *testing.T) {
	h := newHarness(t, nil)

	h.press("enter")
	h.typeText("monday's words")
	h.press("esc")
	h.press("h")

	if got := h.onDisk(aug17); got != "monday's words" {
		t.Errorf("on disk = %q", got)
	}
	if !h.m.page.IsEmpty() {
		t.Errorf("the previous day came back as %q", h.m.page.Body)
	}
}

func TestAutosave(t *testing.T) {
	h := newHarness(t, nil)

	h.press("enter")
	h.typeText("still typing")
	if got := h.onDisk(aug17); got != "" {
		t.Errorf("mori saved mid-sentence: %q", got)
	}

	// The tick that the last keystroke scheduled.
	h.send(autosaveMsg{seq: h.m.saveSeq})

	if got := h.onDisk(aug17); got != "still typing" {
		t.Errorf("on disk = %q, want the autosave to have landed", got)
	}
	if h.m.mode != modeWrite {
		t.Error("autosaving took the cursor out of the page")
	}
}

// A tick from a keystroke that has since been superseded must not save an
// older version of the page.
func TestASupersededAutosaveIsIgnored(t *testing.T) {
	h := newHarness(t, nil)
	h.press("enter")
	h.typeText("one")
	stale := h.m.saveSeq
	h.typeText(" two")

	h.send(autosaveMsg{seq: stale})
	if got := h.onDisk(aug17); got != "" {
		t.Errorf("a stale tick saved %q", got)
	}

	h.send(autosaveMsg{seq: h.m.saveSeq})
	if got := h.onDisk(aug17); got != "one two" {
		t.Errorf("on disk = %q", got)
	}
}

func TestExplicitSaveStaysInThePage(t *testing.T) {
	h := newHarness(t, nil)

	h.press("enter")
	h.typeText("saved on purpose")
	h.press("ctrl+s")

	if got := h.onDisk(aug17); got != "saved on purpose" {
		t.Errorf("on disk = %q", got)
	}
	if h.m.mode != modeWrite {
		t.Error("ctrl+s left the page")
	}
	if !strings.Contains(h.screen(), "saved") {
		t.Errorf("no acknowledgement on screen:\n%s", h.screen())
	}
}

// Emptying a page removes it, rather than leaving a husk on the calendar.
func TestEmptyingAPageRemovesIt(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17: "written by mistake"})

	h.press("enter")
	h.m.ta.SetValue("")
	h.press("esc")

	if has, _ := h.store.Has(aug17); has {
		t.Error("the file survived being emptied")
	}
}

func TestGoToADate(t *testing.T) {
	target := entry.Date{Year: 2019, Month: time.January, Day: 2}
	h := newHarness(t, map[entry.Date]string{target: "a long time ago"})

	h.press("g")
	h.typeText("2 jan 2019")
	h.press("enter")

	if h.m.date != target {
		t.Fatalf("went to %v, want %v", h.m.date, target)
	}
	if !strings.Contains(h.screen(), "a long time ago") {
		t.Errorf("the page isn't on screen:\n%s", h.screen())
	}
}

func TestGoToADateItCannotRead(t *testing.T) {
	h := newHarness(t, nil)

	h.press("g")
	h.typeText("someday")
	h.press("enter")

	if h.m.date != aug17 {
		t.Errorf("an unreadable date moved the day to %v", h.m.date)
	}
	if !strings.Contains(h.screen(), "someday") {
		t.Errorf("mori didn't say what it couldn't read:\n%s", h.screen())
	}
}

func TestGoToCanBeCancelled(t *testing.T) {
	h := newHarness(t, nil)

	h.press("g")
	h.typeText("2019-01-02")
	h.press("esc")

	if h.m.mode != modeRead || h.m.date != aug17 {
		t.Errorf("esc left mode %v on %v", h.m.mode, h.m.date)
	}
}

// Opening mori after a long time away should feel like being welcomed, not
// like being caught.
func TestWelcomeBack(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17.Add(-40): "a while ago"})

	if !strings.Contains(h.screen(), "welcome back") {
		t.Errorf("no welcome after a long gap:\n%s", h.screen())
	}
	// And nothing that counts the days, or mentions the gap at all.
	for _, forbidden := range []string{"40", "days", "streak", "last wrote"} {
		if strings.Contains(strings.ToLower(h.screen()), forbidden) {
			t.Errorf("the greeting mentions %q:\n%s", forbidden, h.screen())
		}
	}
}

func TestNoGreetingWhenYouWereJustHere(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{aug17.Add(-1): "yesterday"})

	if strings.Contains(h.screen(), "welcome back") {
		t.Errorf("greeted after one day away:\n%s", h.screen())
	}
}

func TestHelp(t *testing.T) {
	h := newHarness(t, nil)

	if !strings.Contains(h.screen(), "write") {
		t.Errorf("the footer doesn't say how to write:\n%s", h.screen())
	}

	h.press("?")
	screen := h.screen()
	for _, want := range []string{"previous day", "next day", "new section", "quit"} {
		if !strings.Contains(screen, want) {
			t.Errorf("the help is missing %q:\n%s", want, screen)
		}
	}
}

// A terminal can always be smaller than any layout is willing to be.
func TestTheFrameAlwaysFits(t *testing.T) {
	h := newHarness(t, map[entry.Date]string{
		aug17: strings.Repeat("a very long line that will certainly need wrapping ", 20),
	})

	for _, size := range []struct{ w, h int }{{90, 24}, {40, 10}, {20, 6}, {10, 4}, {120, 40}} {
		h.send(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		lines := strings.Split(h.screen(), "\n")
		if len(lines) > size.h {
			t.Errorf("%dx%d: %d lines", size.w, size.h, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w > size.w {
				t.Errorf("%dx%d: line %d is %d wide", size.w, size.h, i, w)
			}
		}
	}
}
