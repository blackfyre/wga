package components

// AddToItineraryVariant is the validated presentation variant for the typed
// shared itinerary add control. Compact is the small primary-edged chip used
// in dense lists, Row is the 46px inline action row, and Block is the 50px
// full-width record action. Call sites pass the matching typed constant; an
// unknown or empty value is normalised to Compact so it never produces
// arbitrary markup.
type AddToItineraryVariant string

const (
	AddToItineraryCompact AddToItineraryVariant = "compact"
	AddToItineraryRow     AddToItineraryVariant = "row"
	AddToItineraryBlock   AddToItineraryVariant = "block"
)

// normalizeAddToItineraryVariant collapses an unknown or empty variant to the
// compact presentation so an invalid value never renders arbitrary markup.
func normalizeAddToItineraryVariant(variant AddToItineraryVariant) AddToItineraryVariant {
	switch variant {
	case AddToItineraryCompact, AddToItineraryRow, AddToItineraryBlock:
		return variant
	default:
		return AddToItineraryCompact
	}
}

// addToItineraryFormClass returns the wrapper class for the typed variant,
// matching the accepted reference: compact sits inline and self-centres, row
// shrinks beside its action row, and block stretches to the full record width.
func addToItineraryFormClass(variant AddToItineraryVariant) string {
	switch variant {
	case AddToItineraryRow:
		return "shrink-0"
	case AddToItineraryBlock:
		return "w-full"
	default:
		return "shrink-0 self-center"
	}
}

// addToItineraryButtonClass builds the reference button class for the typed
// variant and state. Compact is a small chip; row is a 46px inline action;
// block is a 50px full-width action. Added, full, and available states each
// carry their own border/role treatment.
func addToItineraryButtonClass(variant AddToItineraryVariant, added bool, full bool) string {
	class := "font-mono tracking-[1px] border transition-colors"

	switch variant {
	case AddToItineraryRow:
		class += " text-(length:--t-11) tracking-[1.5px] px-[22px] h-[46px]"
	case AddToItineraryBlock:
		class += " w-full text-xs tracking-[1.5px] px-6 h-[50px]"
	default:
		class += " text-(length:--t-10) px-2.5 py-1.5"
	}

	switch {
	case added:
		class += " border-base-content/20 bg-base-content/6 text-faint-2"
	case full:
		class += " border-base-content/25 text-faint-2"
	case variant == AddToItineraryCompact:
		class += " border-primary text-primary hover:opacity-70"
	default:
		class += " border-control hover:border-primary hover:text-primary"
	}

	return class
}

// addToItineraryButtonLabel returns the reference label for the typed variant
// and state: compact uses short labels, row/block use the full labels, and the
// full state shares a single unavailable label across variants.
func addToItineraryButtonLabel(variant AddToItineraryVariant, added bool, full bool) string {
	if added {
		if variant == AddToItineraryCompact {
			return "ADDED ✓"
		}
		return "IN YOUR ITINERARY ✓"
	}
	if full {
		return "ITINERARY IS FULL"
	}
	if variant == AddToItineraryCompact {
		return "ADD"
	}
	return "ADD TO AN ITINERARY +"
}
