// Package studyboard owns the transient Study Board state contract.
package studyboard

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// MaxWorks is the fixed capacity of one Study Board.
	MaxWorks        = 12
	lookupChunkSize = 200
)

var recordIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)
var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

// Artwork is the deliberately small public projection needed before the
// comparison views are assembled. Unpublished records never enter this type.
type Artwork struct {
	ID         string
	Title      string
	Artist     string
	URL        string
	ImageURL   string
	Date       string
	Dimensions string
	Medium     string
	Location   string
	School     string
	Form       string
	Type       string
}

// Board is one validated, ordered, duplicate-free transient board.
type Board struct {
	Works []Artwork
}

// IDs returns a copy of the board's canonical ordered identifiers.
func (b Board) IDs() []string {
	ids := make([]string, len(b.Works))
	for i, work := range b.Works {
		ids[i] = work.ID
	}
	return ids
}

// AtCapacity reports whether another distinct work can be appended.
func (b Board) AtCapacity() bool {
	return len(b.Works) >= MaxWorks
}

// Resolve validates an untrusted comma-separated board value against currently
// published artwork records while retaining the request order.
func Resolve(app core.App, raw string) (Board, error) {
	candidates := candidateIDs(raw)
	works := make([]Artwork, 0, min(len(candidates), MaxWorks))

	for start := 0; start < len(candidates) && len(works) < MaxWorks; start += lookupChunkSize {
		end := min(start+lookupChunkSize, len(candidates))
		records, err := app.FindRecordsByIds(constants.CollectionArtworks, candidates[start:end])
		if err != nil {
			return Board{}, err
		}

		published := make(map[string]*core.Record, len(records))
		for _, record := range records {
			if record.GetBool("published") {
				published[record.Id] = record
			}
		}

		for _, id := range candidates[start:end] {
			record, ok := published[id]
			if !ok {
				continue
			}
			work, err := projectArtwork(app, record)
			if err != nil {
				return Board{}, err
			}
			works = append(works, work)
			if len(works) == MaxWorks {
				break
			}
		}
	}

	return Board{Works: works}, nil
}

func projectArtwork(app core.App, record *core.Record) (Artwork, error) {
	location, dimensions := locationAndDimensions(record.GetString("comment"))
	if locationIDs := record.GetStringSlice("current_location_id"); len(locationIDs) > 0 {
		currentLocation, err := app.FindRecordById(constants.CollectionLocations, locationIDs[0])
		if err != nil {
			return Artwork{}, err
		}
		if name := strings.TrimSpace(currentLocation.GetString("name")); name != "" {
			location = name
		}
	}
	school, err := relationNames(app, constants.CollectionSchools, record.GetStringSlice("school"))
	if err != nil {
		return Artwork{}, err
	}
	form, err := relationNames(app, constants.CollectionArtForms, record.GetStringSlice("form"))
	if err != nil {
		return Artwork{}, err
	}
	artType, err := relationNames(app, constants.CollectionArtTypes, record.GetStringSlice("type"))
	if err != nil {
		return Artwork{}, err
	}

	title := record.GetString("title")
	workURL := urlutils.GenerateArtworkUrl(urlutils.ArtworkUrlDTO{ArtworkId: record.Id, ArtworkTitle: title})
	artist := ""
	authorIDs := record.GetStringSlice("author")
	if len(authorIDs) > 0 {
		authorRecord, findErr := app.FindRecordById(constants.CollectionArtists, authorIDs[0])
		if findErr == nil && authorRecord.GetBool("published") {
			artist = authorRecord.GetString("filing_name")
			workURL = urlutils.GenerateFullArtworkUrl(urlutils.ArtworkUrlDTO{
				ArtistId: authorRecord.Id, ArtistName: authorRecord.GetString("name"),
				ArtworkId: record.Id, ArtworkTitle: title,
			})
		}
	}

	medium := strings.TrimSpace(record.GetString("technique"))
	if dimensions != "" {
		medium = strings.TrimSpace(strings.TrimSuffix(medium, ", "+dimensions))
	}
	imageURL := ""
	if record.GetString("image") != "" {
		imageURL = urlutils.GenerateArtworkImageURL(record, urlutils.DeliveryProfileRelatedTimelineCard, "")
	}

	return Artwork{
		ID: record.Id, Title: title, Artist: artist, URL: workURL, ImageURL: imageURL,
		Date: artworkDate(record), Dimensions: dimensions, Medium: medium,
		Location: location, School: school, Form: form, Type: artType,
	}, nil
}

func relationNames(app core.App, collection string, ids []string) (string, error) {
	if len(ids) == 0 {
		return "", nil
	}
	records, err := app.FindRecordsByIds(collection, ids)
	if err != nil {
		return "", err
	}
	byID := make(map[string]string, len(records))
	for _, record := range records {
		byID[record.Id] = strings.TrimSpace(record.GetString("name"))
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name := byID[id]; name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", "), nil
}

func artworkDate(record *core.Record) string {
	start := record.GetInt("date_start")
	if start <= 0 {
		start = record.GetInt("year")
	}
	if start <= 0 {
		return ""
	}
	end := record.GetInt("date_end")
	if end <= 0 {
		end = start
	}
	value := strconv.Itoa(start)
	if end != start {
		value += "–" + strconv.Itoa(end)
	}
	if qualifier := strings.TrimSpace(record.GetString("date_qualifier")); qualifier != "" {
		return qualifier + " " + value
	}
	if record.GetBool("is_circa") {
		return "circa " + value
	}
	return value
}

func locationAndDimensions(comment string) (string, string) {
	plain := html.UnescapeString(htmlTagPattern.ReplaceAllString(comment, " "))
	parts := strings.Split(plain, " · ")
	if len(parts) < 3 {
		return "", ""
	}
	return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])
}

func candidateIDs(raw string) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		id := strings.TrimSpace(value)
		if !recordIDPattern.MatchString(id) {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
