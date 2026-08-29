package artworks

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/pocketbase/pocketbase/core"
)

var paletteHex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Palette returns the sampled palette persisted with an artwork. Source colour
// names are optional; the displayed hex value remains the truthful identifier
// when the producer did not supply one.
func Palette(artwork *core.Record) []dto.ColourSwatch {
	data, err := json.Marshal(artwork.Get("colour_palette"))
	if err != nil {
		return nil
	}

	var bands []dto.ColourSwatch
	if err := json.Unmarshal(data, &bands); err != nil {
		return nil
	}

	for index := range bands {
		bands[index].Name = strings.TrimSpace(bands[index].Name)
		if !paletteHex.MatchString(bands[index].Hex) || bands[index].Weight <= 0 {
			return nil
		}
	}

	return bands
}
