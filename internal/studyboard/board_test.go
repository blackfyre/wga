package studyboard

import (
	"fmt"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestCandidateIDsOmitInvalidAndDuplicateValues(t *testing.T) {
	got := candidateIDs(" work0000000001,invalid id,work0000000001,,work0000000002 ")
	want := []string{"work0000000001", "work0000000002"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("candidateIDs() = %v, want %v", got, want)
	}
}

func TestArtworkDateUsesQualifiedSourceSpanWithoutFabrication(t *testing.T) {
	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(
		&core.NumberField{Name: "year"},
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "date_end"},
		&core.BoolField{Name: "is_circa"},
		&core.TextField{Name: "date_qualifier"},
	)
	record := core.NewRecord(collection)
	record.Set("date_start", 1500)
	record.Set("date_end", 1510)
	record.Set("is_circa", true)
	if got := artworkDate(record); got != "circa 1500–1510" {
		t.Errorf("artworkDate() = %q", got)
	}
	record.Set("date_qualifier", "after")
	if got := artworkDate(record); got != "after 1500–1510" {
		t.Errorf("qualified artworkDate() = %q", got)
	}
}

func TestLocationAndDimensionsOnlyReturnsSourceBackedValues(t *testing.T) {
	location, dimensions := locationAndDimensions("<p>Oil on panel · Rijksmuseum &amp; Gallery · 50 × 40 cm</p>")
	if location != "Rijksmuseum & Gallery" || dimensions != "50 × 40 cm" {
		t.Errorf("locationAndDimensions() = %q, %q", location, dimensions)
	}
	location, dimensions = locationAndDimensions("No structured catalogue summary")
	if location != "" || dimensions != "" {
		t.Errorf("unstructured values = %q, %q, want empty", location, dimensions)
	}
}

func TestBoardCapacity(t *testing.T) {
	works := make([]Artwork, MaxWorks)
	for i := range works {
		works[i] = Artwork{ID: fmt.Sprintf("work%011d", i)}
	}
	board := Board{Works: works}

	if !board.AtCapacity() {
		t.Fatal("AtCapacity() = false, want true at twelve works")
	}
	ids := board.IDs()
	ids[0] = "changed"
	if board.Works[0].ID == "changed" {
		t.Fatal("IDs() returned mutable board storage")
	}
}
