package pages

import "github.com/blackfyre/wga/internal/assets/templ/dto"

// ArtworkSearchFacet is the server-owned summary and disclosure state for one
// progressively enhanced artwork-search facet.
type ArtworkSearchFacet struct {
	Label   string
	Summary string
	Active  bool
	Open    bool
	Last    bool
}

// ArtworkSearchCollectionOption is one counted collection holding. Options are
// already ordered and bounded by the artwork-search query.
type ArtworkSearchCollectionOption struct {
	Label    string
	Value    string
	Count    int
	Selected bool
}

// ArtworkSearchCollectionFacet carries the collection option search and its
// truthful bounded-query disclosure. VenueQuery changes only this option set;
// it is not a result filter.
type ArtworkSearchCollectionFacet struct {
	Facet           ArtworkSearchFacet
	Name            string
	QueryField      dto.Field
	Options         []ArtworkSearchCollectionOption
	Note            string
	TotalOptions    int
	OmittedOptions  int
	OmittedHoldings int
}

// ArtworkSearchFacets is the presentation-only facet contract. ActiveCount and
// every summary/open flag are derived on the server from canonical filter state.
type ArtworkSearchFacets struct {
	ActiveCount int
	Query       ArtworkSearchFacet
	Technique   ArtworkSearchFacet
	School      ArtworkSearchFacet
	Form        ArtworkSearchFacet
	Type        ArtworkSearchFacet
	Period      ArtworkSearchFacet
	Collection  ArtworkSearchCollectionFacet
	Year        ArtworkSearchFacet
	YearRange   dto.RangeField
}
