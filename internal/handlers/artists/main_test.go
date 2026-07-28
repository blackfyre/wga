package artists

import "testing"

func TestBuildArtistsFilter(t *testing.T) {
	filter, params, activeLetter := buildArtistsFilter("Vermeer", " v ")

	if activeLetter != "V" {
		t.Fatalf("activeLetter = %q, want V", activeLetter)
	}
	if filter != "published = true && name ~ {:searchExpression} && name ~ {:activeLetter}" {
		t.Fatalf("filter = %q", filter)
	}
	if params["searchExpression"] != "Vermeer" || params["activeLetter"] != "V%" {
		t.Fatalf("params = %#v", params)
	}
}

func TestBuildArtistsFilterIgnoresInvalidLetter(t *testing.T) {
	filter, params, activeLetter := buildArtistsFilter("", "12")

	if activeLetter != "" {
		t.Fatalf("activeLetter = %q, want empty", activeLetter)
	}
	if filter != "published = true" {
		t.Fatalf("filter = %q", filter)
	}
	if params["searchExpression"] != "" {
		t.Fatalf("params = %#v", params)
	}
}
