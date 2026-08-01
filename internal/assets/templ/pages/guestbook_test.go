package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
