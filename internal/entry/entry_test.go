package entry

import (
	"reflect"
	"testing"
	"time"
)

func page(body string) Entry {
	return Entry{Date: date(2026, time.August, 17), Body: body}
}

func TestTags(t *testing.T) {
	tests := []struct {
		body string
		want []string
	}{
		{"nothing here", nil},
		{"a good day for #go", []string{"go"}},
		{"#go and #Go and #GO", []string{"go"}},
		{"#photography, then #go.", []string{"photography", "go"}},
		{"tags keep their order: #zebra #apple", []string{"zebra", "apple"}},
		{"#side-project and #_private are fine", []string{"side-project", "_private"}},

		// The things that must not become tags.
		{"## 23:04", nil},
		{"# A Markdown heading", nil},
		{"### Another", nil},
		{"issue #4 is fixed", []string{"4"}}, // a digit-led tag is still a tag
		{"C# is a language", nil},            // no space before the hash
		{"a#b", nil},
	}

	for _, tt := range tests {
		got := page(tt.body).Tags()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Tags(%q) = %v, want %v", tt.body, got, tt.want)
		}
	}
}

func TestHasTag(t *testing.T) {
	e := page("edited the #photography photos")
	if !e.HasTag("photography") {
		t.Error("HasTag(photography) = false")
	}
	if !e.HasTag("#Photography") {
		t.Error("HasTag normalises its argument")
	}
	if e.HasTag("go") {
		t.Error("HasTag(go) = true")
	}
	if e.HasTag("") {
		t.Error("HasTag(\"\") = true")
	}
}

func TestSections(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []Section
	}{
		{
			name: "a day written in one sitting has no timestamps",
			body: "Slow morning.\n\nCoffee on the balcony.",
			want: []Section{{At: "", Body: "Slow morning.\n\nCoffee on the balcony."}},
		},
		{
			name: "coming back later",
			body: "Slow morning.\n\n## 23:04\n\nWent back to it after dinner.",
			want: []Section{
				{At: "", Body: "Slow morning."},
				{At: "23:04", Body: "Went back to it after dinner."},
			},
		},
		{
			name: "a page that opens with a heading has no empty first block",
			body: "## 09:14\n\nUp early.",
			want: []Section{{At: "09:14", Body: "Up early."}},
		},
		{
			name: "times are normalised",
			body: "## 9:04\n\nx",
			want: []Section{{At: "09:04", Body: "x"}},
		},
		{
			name: "a stamped section you left empty is still a section",
			body: "morning\n\n## 23:04\n",
			want: []Section{{At: "", Body: "morning"}, {At: "23:04", Body: ""}},
		},
		{
			name: "prose headings are prose",
			body: "## Things I did\n\n- went to the gym",
			want: []Section{{At: "", Body: "## Things I did\n\n- went to the gym"}},
		},
		{
			name: "so is a heading with an impossible time",
			body: "## 41:99\n\nx",
			want: []Section{{At: "", Body: "## 41:99\n\nx"}},
		},
		{
			name: "and so is a hash with no space",
			body: "##23:04\n\nx",
			want: []Section{{At: "", Body: "##23:04\n\nx"}},
		},
		{
			name: "an empty page has no sections",
			body: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := page(tt.body).Sections()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sections() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSectionHeading(t *testing.T) {
	at := time.Date(2026, time.August, 17, 9, 4, 0, 0, time.UTC)
	if got := SectionHeading(at); got != "## 09:04" {
		t.Errorf("SectionHeading = %q", got)
	}
	// What mori writes, mori must be able to read back.
	if v, ok := SectionAt(SectionHeading(at)); !ok || v != "09:04" {
		t.Errorf("SectionAt(SectionHeading(...)) = %q, %v", v, ok)
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		body string
		want bool
	}{
		{"", true},
		{"   \n\n\t\n", true},
		{"## 09:14\n\n## 23:04\n", true}, // opened, stamped, never written in
		{"x", false},
		{"## 09:14\n\nx", false},
	}
	for _, tt := range tests {
		if got := page(tt.body).IsEmpty(); got != tt.want {
			t.Errorf("IsEmpty(%q) = %v, want %v", tt.body, got, tt.want)
		}
	}
}

func TestWords(t *testing.T) {
	e := page("Slow morning.\n\n## 23:04\n\nWent back to it after dinner.")
	if got := e.Words(); got != 8 {
		t.Errorf("Words = %d, want 8 (section headings don't count)", got)
	}
}

func TestExcerpt(t *testing.T) {
	tests := []struct {
		body  string
		width int
		want  string
	}{
		{"", 20, ""},
		{"a quiet day", 20, "a quiet day"},
		{"## 09:14\n\na quiet day", 20, "a quiet day"},
		{"\n\n  a quiet day  ", 20, "a quiet day"},
		{"# A title\n\nbody", 20, "A title"},
		{"- a bullet", 20, "a bullet"},
		{"> a quote", 20, "a quote"},
		{"a much longer line than fits", 12, "a much long…"},
		{"## 09:14\n\n## 23:04\n", 20, ""},
	}
	for _, tt := range tests {
		if got := page(tt.body).Excerpt(tt.width); got != tt.want {
			t.Errorf("Excerpt(%q, %d) = %q, want %q", tt.body, tt.width, got, tt.want)
		}
	}
}

func TestNormalizeTag(t *testing.T) {
	tests := map[string]string{
		"#Photography": "photography",
		"  #GO  ":      "go",
		"side-project": "side-project",
		"with spaces":  "withspaces",
		"":             "",
		"#":            "",
		"###":          "",
	}
	for in, want := range tests {
		if got := NormalizeTag(in); got != want {
			t.Errorf("NormalizeTag(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeMood(t *testing.T) {
	tests := map[string]string{
		"Calm":            "calm",
		"  TIRED ":        "tired",
		"quietly hopeful": "quietly", // one word is the whole feature
		"7/10":            "",
		"":                "",
	}
	for in, want := range tests {
		if got := NormalizeMood(in); got != want {
			t.Errorf("NormalizeMood(%q) = %q, want %q", in, got, want)
		}
	}
}
