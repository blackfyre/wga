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
