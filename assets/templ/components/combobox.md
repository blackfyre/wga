# Combobox Component Usage Examples

## 1. Basic Usage

The combobox component is a reusable, filterable single select dropdown that looks like a text field but returns structured value/label pairs.

### Data Structure

The combobox expects options in this format:

```go
type ComboboxOption struct {
    Value string // The value returned when selected (e.g., URL, ID)
    Label string // The display text shown to users
}
```

### Basic Example

```templ
@dto.Combobox(dto.ComboboxProps{
    ID:          "artist-select",
    Name:        "artist",
    Placeholder: "Search for an artist...",
    Options: []dto.ComboboxOption{
        {Value: "/artist/1", Label: "Leonardo da Vinci"},
        {Value: "/artist/2", Label: "Michelangelo"},
        {Value: "/artist/3", Label: "Raphael"},
    },
    Value:    "", // Current selected value
    Required: false,
})
```

## 2. Artwork Search Integration

### Current Implementation (datalist)

The artworks search currently uses a datalist:

```templ
<label class="input input-bordered flex items-center gap-2">
    Artist
    <input
        class="grow"
        type="search"
        list="artist_list"
        id="artist_name"
        autocomplete="artist_name"
        name="artist"
        value={ b.ActiveFilterValues.ArtistString }
    />
    <datalist id="artist_list">
        for _, v := range b.ArtistNameList {
            <option value={ v }></option>
        }
    </datalist>
</label>
```

### Updated Implementation (combobox)

First, create a helper function to convert the artist name list:

```go
// In your handler or utility package
func ConvertArtistMapToComboboxOptions(artistMap map[string]string) []dto.ComboboxOption {
    options := make([]dto.ComboboxOption, 0, len(artistMap))
    for url, name := range artistMap {
        options = append(options, dto.ComboboxOption{
            Value: url,   // The artist URL for programmatic use
            Label: name,  // The artist name for display
        })
    }
    // Sort by label for better UX
    sort.Slice(options, func(i, j int) bool {
        return options[i].Label < options[j].Label
    })
    return options
}
```

Then use it in your template:

```templ
<label class="form-control w-full">
    Artist
    @dto.Combobox(dto.ComboboxProps{
        ID:          "artist_select",
        Name:        "artist",
        Placeholder: "Search for an artist...",
        Options:     ConvertArtistMapToComboboxOptions(b.ArtistNameList),
        Value:       b.ActiveFilterValues.ArtistString,
        Size:        "md",
        Color:       "bordered",
        HxGet:       "/artworks/results",
        HxTarget:    b.HxTarget,
        HxTrigger:   "change",
    })
</label>
```

## 3. Advanced Usage with HTMX

### Form Integration

```templ
<form
    hx-get="/artworks/results"
    hx-target="#results"
    hx-disabled-elt="input,button,select"
>
    @dto.Combobox(dto.ComboboxProps{
        ID:          "category-select",
        Name:        "category",
        Placeholder: "Select a category...",
        Options: []dto.ComboboxOption{
            {Value: "painting", Label: "Paintings"},
            {Value: "sculpture", Label: "Sculptures"},
            {Value: "drawing", Label: "Drawings"},
        },
        Required: true,
        Size:     "lg",
        Color:    "primary",
    })

    <button type="submit" class="btn btn-primary">Search</button>
</form>
```

### Dynamic Loading

```templ
@dto.Combobox(dto.ComboboxProps{
    ID:          "dynamic-select",
    Name:        "selection",
    Placeholder: "Type to search...",
    Options:     []dto.ComboboxOption{}, // Empty, will be populated
    HxGet:       "/api/search",
    HxTarget:    "#dynamic-select-dropdown",
    HxTrigger:   "keyup changed delay:300ms",
})
```

## 4. Styling Options

### Sizes

- `xs` - Extra small
- `sm` - Small
- `md` - Medium (default)
- `lg` - Large
- `xl` - Extra large

### Colors

- `primary` - Primary brand color
- `secondary` - Secondary brand color
- `accent` - Accent color
- `neutral` - Neutral color
- `bordered` - Default bordered style
- `ghost` - Ghost style

### Custom Classes

```templ
@dto.Combobox(dto.ComboboxProps{
    ID:    "custom-select",
    Name:  "custom",
    Class: "custom-class another-class",
    // ... other props
})
```

## 5. Accessibility Features

The combobox includes built-in accessibility features:

- Proper ARIA attributes
- Keyboard navigation (arrows, enter, escape)
- Screen reader support
- Focus management

## 6. JavaScript Integration

The component automatically initializes when:

- The page loads
- HTMX loads new content
- You call `combobox()` manually

### Manual Initialization

```javascript
// Initialize all comboboxes on the page
combobox();

// Or initialize after HTMX content loads
htmx.on("htmx:afterSwap", function () {
  combobox();
});
```

## 7. Benefits over HTML Datalist

1. **Better Styling Control**: Full daisyUI integration
2. **Programmatic Values**: Returns structured data (URL) while displaying user-friendly labels
3. **Better Filtering**: More responsive and customizable search
4. **Accessibility**: Better screen reader and keyboard support
5. **Mobile Support**: Works consistently across devices
6. **HTMX Integration**: Seamless integration with HTMX workflows

## 8. Migration Guide

### From HTML Select

Replace:

```html
<select name="artist" class="select select-bordered">
  <option value="/artist/1">Leonardo da Vinci</option>
  <option value="/artist/2">Michelangelo</option>
</select>
```

With:

```templ
@dto.Combobox(dto.ComboboxProps{
    Name: "artist",
    Options: []dto.ComboboxOption{
        {Value: "/artist/1", Label: "Leonardo da Vinci"},
        {Value: "/artist/2", Label: "Michelangelo"},
    },
})
```

### From HTML Datalist

Replace:

```html
<input list="artists" name="artist" />
<datalist id="artists">
  <option value="Leonardo da Vinci"></option>
  <option value="Michelangelo"></option>
</datalist>
```

With:

```templ
@dto.Combobox(dto.ComboboxProps{
    Name: "artist",
    Options: []dto.ComboboxOption{
        {Value: "/artist/1", Label: "Leonardo da Vinci"},
        {Value: "/artist/2", Label: "Michelangelo"},
    },
})
```
