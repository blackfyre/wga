package seed

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// validArtworksContractCollection builds the artworks collection with the
// concrete field types and relation cardinality the importer expects.
func validArtworksContractCollection() *core.Collection {
	collection := core.NewBaseCollection("Artworks")
	collection.Id = "artworks"
	collection.Fields.Add(
		&core.NumberField{Name: "source_row"},
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "date_end"},
		&core.BoolField{Name: "is_circa"},
		&core.TextField{Name: "date_qualifier"},
		&core.TextField{Name: "timeframe_text"},
		&core.RelationField{Name: "current_location_id", CollectionId: "locations", MinSelect: 0, MaxSelect: 1},
		&core.RelationField{Name: "art_period_id", CollectionId: "art_periods", MinSelect: 0, MaxSelect: 1},
		&core.TextField{Name: "source_url"},
		&core.TextField{Name: "source_path"},
		&core.TextField{Name: "source_comment"},
		&core.JSONField{Name: "colour_palette"},
		&core.JSONField{Name: "colour_signature"},
		&core.TextField{Name: "colour_profile_version"},
		&core.TextField{Name: "colour_image_hash"},
		&core.NumberField{Name: "image_size_bytes"},
	)
	return collection
}

// validSelectionsContractCollection builds the art_selections collection with
// the relation targets and cardinality the importer and read-model expect.
func validSelectionsContractCollection() *core.Collection {
	collection := core.NewBaseCollection("Art_selections")
	collection.Id = "art_selections"
	collection.Fields.Add(
		&core.RelationField{Name: "artist", CollectionId: "artists", MinSelect: 1, MaxSelect: 1},
		&core.RelationField{Name: "artworks", CollectionId: "artworks", MinSelect: 1, MaxSelect: 1000},
	)
	return collection
}

func TestValidateSourceFieldContractsAcceptsBaseline(t *testing.T) {
	artworks := validArtworksContractCollection()
	selections := validSelectionsContractCollection()

	if err := validateSourceFieldContracts(artworks, selections); err != nil {
		t.Fatalf("expected baseline contracts to pass, got %v", err)
	}
}

func TestValidateSourceFieldContractsRejectsWrongScalarType(t *testing.T) {
	artworks := validArtworksContractCollection()
	artworks.Fields.RemoveByName("source_row")
	artworks.Fields.Add(&core.TextField{Name: "source_row"})
	selections := validSelectionsContractCollection()

	err := validateSourceFieldContracts(artworks, selections)
	if err == nil || !strings.Contains(err.Error(), "source_row") {
		t.Fatalf("expected wrong scalar type error for source_row, got %v", err)
	}
}

func TestValidateSourceFieldContractsRejectsWrongRelationCollection(t *testing.T) {
	artworks := validArtworksContractCollection()
	artworks.Fields.RemoveByName("current_location_id")
	artworks.Fields.Add(&core.RelationField{Name: "current_location_id", CollectionId: "schools", MinSelect: 0, MaxSelect: 1})
	selections := validSelectionsContractCollection()

	err := validateSourceFieldContracts(artworks, selections)
	if err == nil || !strings.Contains(err.Error(), "targets") {
		t.Fatalf("expected wrong relation collection error, got %v", err)
	}
}

func TestValidateSourceFieldContractsRejectsWrongCardinality(t *testing.T) {
	artworks := validArtworksContractCollection()
	artworks.Fields.RemoveByName("art_period_id")
	artworks.Fields.Add(&core.RelationField{Name: "art_period_id", CollectionId: "art_periods", MinSelect: 0, MaxSelect: 10})
	selections := validSelectionsContractCollection()

	err := validateSourceFieldContracts(artworks, selections)
	if err == nil || !strings.Contains(err.Error(), "cardinality") {
		t.Fatalf("expected wrong cardinality error, got %v", err)
	}
}

func TestValidateSourceFieldContractsRejectsWrongSelectionRelation(t *testing.T) {
	artworks := validArtworksContractCollection()

	wrongTarget := validSelectionsContractCollection()
	wrongTarget.Fields.RemoveByName("artist")
	wrongTarget.Fields.Add(&core.RelationField{Name: "artist", CollectionId: "schools", MinSelect: 1, MaxSelect: 1})
	if err := validateSourceFieldContracts(artworks, wrongTarget); err == nil || !strings.Contains(err.Error(), "targets") {
		t.Fatalf("expected wrong selection relation target error, got %v", err)
	}

	wrongCardinality := validSelectionsContractCollection()
	wrongCardinality.Fields.RemoveByName("artworks")
	wrongCardinality.Fields.Add(&core.RelationField{Name: "artworks", CollectionId: "artworks", MinSelect: 0, MaxSelect: 10})
	if err := validateSourceFieldContracts(artworks, wrongCardinality); err == nil || !strings.Contains(err.Error(), "cardinality") {
		t.Fatalf("expected wrong selection cardinality error, got %v", err)
	}
}
