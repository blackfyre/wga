package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestAddToStudyBoardActionCarriesPublishedProjectionOutsideLinks(t *testing.T) {
	var output bytes.Buffer
	if err := AddToStudyBoardAction("work00000000001", "A & B", "/image.jpg", AddToStudyBoardBlock).Render(context.Background(), &output); err != nil {
		t.Fatalf("render action: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{`type="button"`, `data-study-board-add="work00000000001"`, `data-study-board-title="A &amp; B"`, `data-study-board-image="/image.jpg"`, "ADD TO STUDY BOARD +", "hover:border-wga-accent", "hover:bg-wga-accent-tint"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("action missing %q", expected)
		}
	}
	if strings.Contains(rendered, "<a ") {
		t.Fatal("Study Board action must not be an enclosing or nested record link")
	}
}

func TestAddToStudyBoardActionUsesReferenceRanks(t *testing.T) {
	for _, test := range []struct {
		name      string
		variant   AddToStudyBoardVariant
		expected  string
		forbidden string
	}{
		{"block", AddToStudyBoardBlock, "block min-h-12 w-full px-4 py-2.5", "leading-[17px]"},
		{"row", AddToStudyBoardRow, "min-h-12 px-3.5 py-2", "leading-[17px]"},
		{"card", AddToStudyBoardCard, "px-2.5 py-[7px] leading-[17px]", "min-h-12"},
		{"compact", AddToStudyBoardCompact, "px-2.5 py-[7px] leading-[17px]", "min-h-12"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := AddToStudyBoardAction("work00000000001", "Work", "/image.jpg", test.variant).Render(context.Background(), &output); err != nil {
				t.Fatalf("render action: %v", err)
			}
			rendered := output.String()
			if !strings.Contains(rendered, test.expected) {
				t.Errorf("action missing rank %q", test.expected)
			}
			if strings.Contains(rendered, test.forbidden) {
				t.Errorf("action unexpectedly contains %q", test.forbidden)
			}
		})
	}
}

func TestStudyBoardShelfRegistersAboveItineraryInMeasuredStack(t *testing.T) {
	works := []dto.StudyBoardWork{{ID: "one", Title: "First", ImageURL: "/first.jpg"}, {ID: "two", Title: "Second", ImageURL: "/second.jpg"}}
	var output bytes.Buffer
	if err := StudyBoardShelf(works).Render(context.Background(), &output); err != nil {
		t.Fatalf("render shelf: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{`data-study-board-ids="one,two"`, `data-wga-bottom-stack-order="20"`, "STUDY BOARD · 2 OF 12", "First · Second", `href="/study-board?board=one,two"`, "OPEN BOARD →", "CLEAR"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("shelf missing %q", expected)
		}
	}
}
