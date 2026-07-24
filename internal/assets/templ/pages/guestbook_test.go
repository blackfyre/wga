package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestGuestbookBlockOnlyPromptsToSignForCurrentYear(t *testing.T) {
	tests := []struct {
		name         string
		selectedYear string
		currentYear  string
		wantPrompt   bool
	}{
		{
			name:         "current year",
			selectedYear: "2026",
			currentYear:  "2026",
			wantPrompt:   true,
		},
		{
			name:         "past year",
			selectedYear: "2025",
			currentYear:  "2026",
			wantPrompt:   false,
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

			prompt := "Be the first to sign the guestbook for " + tt.selectedYear + "!"
			if got := strings.Contains(output.String(), prompt); got != tt.wantPrompt {
				t.Errorf("sign prompt present = %t, want %t", got, tt.wantPrompt)
			}
		})
	}
}
