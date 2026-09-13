package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestStudyBoardValuesSameRequiresKnownEqualComparison(t *testing.T) {
	for _, test := range []struct {
		values []string
		want   bool
	}{
		{[]string{"Oil", "Oil"}, true},
		{[]string{"Oil", ""}, false},
		{[]string{"Oil", "Tempera"}, false},
		{[]string{"Oil"}, false},
	} {
		if got := studyBoardValuesSame(test.values); got != test.want {
			t.Errorf("studyBoardValuesSame(%v) = %t, want %t", test.values, got, test.want)
		}
	}
}

func TestStudyBoardPageRendersSevenFieldsBothViewsAndNativeControls(t *testing.T) {
	view := StudyBoardView{Works: []dto.StudyBoardWork{
		{ID: "work00000000001", Title: "First", URL: "/first", Date: "1500", Medium: "Oil", Type: "Painting"},
		{ID: "work00000000002", Title: "Second", URL: "/second", Date: "1500", Medium: "", Type: "Painting"},
	}, IDs: []string{"work00000000001", "work00000000002"}, HasURLState: true}
	var output bytes.Buffer
	if err := StudyBoardPage(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{"MATRIX", "BOARD", "DATE", "DIMENSIONS", "MEDIUM", "LOCATION", "SCHOOL", "FORM", "TYPE", `data-study-board-move="earlier"`, `data-study-board-move="later"`, `data-study-board-remove="work00000000001"`, `href="/itineraries/draft/from-board?board=work00000000001,work00000000002"`, "MAKE ITINERARY →"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("page missing %q", expected)
		}
	}
	if got := strings.Count(rendered, ">SAME</span>"); got != 2 {
		t.Errorf("SAME markers = %d, want 2 for known equal Date and Type only", got)
	}
}
