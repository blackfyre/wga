package studyboard

import (
	"fmt"
	"testing"
)

func TestCandidateIDsOmitInvalidAndDuplicateValues(t *testing.T) {
	got := candidateIDs(" work0000000001,invalid id,work0000000001,,work0000000002 ")
	want := []string{"work0000000001", "work0000000002"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("candidateIDs() = %v, want %v", got, want)
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
