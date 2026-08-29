package artworks

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/pocketbase/pocketbase/core"
)

var paletteHex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Palette returns the source-named sampled palette persisted with an artwork.
// An incomplete or malformed source profile is omitted rather than presented as
// a colour name the archive did not supply.
func Palette(artwork *core.Record) []dto.ColourSwatch {
	data, err := json.Marshal(artwork.Get("colour_palette"))
	if err != nil {
		return nil
	}

	var bands []dto.ColourSwatch
	if err := json.Unmarshal(data, &bands); err != nil {
		return nil
	}

	for _, band := range bands {
		if strings.TrimSpace(band.Name) == "" || !paletteHex.MatchString(band.Hex) || band.Weight <= 0 {
			return nil
		}
	}

	return bands
}
