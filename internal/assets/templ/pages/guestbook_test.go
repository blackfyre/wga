package pages

import (
	"bytes"
	"context"
	"html"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestGuestbookBlockAlwaysRendersSigningLink(t *testing.T) {
	tests := []struct {
		name         string
		selectedYear string
		currentYear  string
		wantSignLink bool
	}{
		{
			name:         "current year",
			selectedYear: "2026",
			currentYear:  "2026",
			wantSignLink: true,
		},
		{
			name:         "past year",
			selectedYear: "2025",
			currentYear:  "2026",
			wantSignLink: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			content := GuestbookView{
				SelectedYear: tt.selectedYear,
				CurrentYear:  tt.currentYear,
			}

			if err := GuestbookBlock(content).Render(context.Background(), &output); err != nil {
				t.Fatalf("render guestbook block: %v", err)
			}

			if got := strings.Contains(output.String(), "SIGN THE GUESTBOOK →"); got != tt.wantSignLink {
				t.Errorf("sign link present = %t, want %t", got, tt.wantSignLink)
			}
		})
	}
}

func TestGuestbookFormSupportsNoJavaScriptWithoutCollectingEmail(t *testing.T) {
	var output bytes.Buffer
	if err := GuestbookEntryForm(GuestbookFormView{}).Render(context.Background(), &output); err != nil {
		t.Fatalf("render guestbook form: %v", err)
	}

	markup := output.String()
	for _, required := range []string{
		`action="/guestbook/add"`,
		`method="post"`,
		`name="sender_name"`,
		`name="location"`,
		`name="message"`,
		`AN APPROVED NOTE BECOMES A PUBLIC ARCHIVE RECORD WHILE APPROVAL REMAINS IN FORCE.`,
		`UNREVIEWED AND REJECTED NOTES EXPIRE AFTER 90 DAYS.`,
		`href="/pages/privacy-policy"`,
	} {
		if !strings.Contains(markup, required) {
			t.Errorf("form does not contain %q", required)
		}
	}
	if strings.Contains(markup, `name="sender_email"`) {
		t.Fatal("guestbook form collects an email address")
	}
}

func TestGuestbookEntryEscapesMessageAndRendersHistoricalMetadata(t *testing.T) {
	entry := dto.GuestbookEntry{
		Name:     "Jane",
		Location: "Delft",
		Created:  "2025-06-02",
		Message:  `<script>alert("private")</script>`,
	}
	var output bytes.Buffer
	if err := GuestbookEntry(entry, 0).Render(context.Background(), &output); err != nil {
		t.Fatalf("render guestbook entry: %v", err)
	}

	markup := output.String()
	if strings.Contains(markup, "<script>") {
		t.Fatal("guestbook message rendered as trusted HTML")
	}
	if !strings.Contains(markup, html.EscapeString(entry.Message)) {
		t.Fatalf("escaped message missing: %s", markup)
	}
	for _, required := range []string{"Jane", "Delft", `<time datetime="2025-06-02">2025-06-02</time>`, `data-kbd-idx="0"`} {
		if !strings.Contains(markup, required) {
			t.Errorf("entry does not contain %q", required)
		}
	}
}

func TestGuestbookURLPreservesSearchYearAndBoundedShow(t *testing.T) {
	view := GuestbookView{Query: "blue chapel"}
	got := guestbookURL(view, "2025", 20)
	if got != "/guestbook?q=blue+chapel&show=20&year=2025" {
		t.Fatalf("guestbook URL = %q", got)
	}
}

func TestGuestbookBlockAlwaysShowsYearNavigation(t *testing.T) {
	tests := []struct {
		name         string
		yearOptions  []string
			wantSelector bool
	}{
		{
			name:         "no years",
				wantSelector: true,
		},
		{
			name:         "one year",
			yearOptions:  []string{"2020"},
				wantSelector: true,
		},
		{
			name:         "multiple years",
			yearOptions:  []string{"2026", "2020"},
			wantSelector: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			content := GuestbookView{
				YearOptions: tt.yearOptions,
			}

			if err := GuestbookBlock(content).Render(context.Background(), &output); err != nil {
				t.Fatalf("render guestbook block: %v", err)
			}

			if got := strings.Contains(output.String(), `aria-label="Filter guestbook entries by year"`); got != tt.wantSelector {
				t.Errorf("year navigation present = %t, want %t", got, tt.wantSelector)
			}
		})
	}
}

func TestGuestbookBlockRendersSharedPageHeadOnce(t *testing.T) {
	content := GuestbookView{
		Total:        5,
		ScopeTotal:   5,
		SelectedYear: "all",
		YearOptions:  []string{"2026"},
	}
	var output bytes.Buffer
	if err := GuestbookBlock(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render guestbook block: %v", err)
	}
	rendered := output.String()

	for _, expected := range []string{
		"10 — GUESTBOOK",
		">Guestbook</h1>",
		"Notes from people who have spent time with the collection. 5 entries in the archive.",
		"text-(length:--t-11)",
		"text-muted",
		"text-(length:--t-32)",
		"md:text-(length:--t-44)",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("guestbook page head does not contain %q", expected)
		}
	}
	if got := strings.Count(rendered, "<h1"); got != 1 {
		t.Errorf("h1 count = %d, want 1", got)
	}
	for _, expected := range []string{
		`hx-get="/guestbook"`,
		`hx-target="#guestbook"`,
		`aria-label="Filter guestbook entries by year"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("guestbook filters do not contain %q", expected)
		}
	}
}
