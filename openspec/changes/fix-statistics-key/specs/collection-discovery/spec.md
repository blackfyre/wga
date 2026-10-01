## ADDED Requirements

### Requirement: Statistics keys match their charts in every palette
The system SHALL render each Statistics chart key with exactly one swatch per series. Each swatch SHALL show the colour the chart uses for that series under the active palette and theme. A series that groups unattributed values ("Other") SHALL draw as a hatch in both the key and the chart. The chart SHALL draw no second legend of its own, and SHALL redraw with the new colours when the visitor changes the palette or the theme.

#### Scenario: Visitor reads the art-form key
- **WHEN** a visitor opens the Statistics route in any palette and theme
- **THEN** each art-form key row shows one swatch, and its colour equals the colour of the matching donut segment

#### Scenario: Visitor reads the school key
- **WHEN** a visitor opens the Statistics route with JavaScript available
- **THEN** each school key swatch matches the colour of that school's stacked segments, and "Other" is a hatch in both

#### Scenario: Visitor changes the palette or the theme
- **WHEN** a visitor changes the palette or the theme while the Statistics page is open
- **THEN** the charts redraw, and each key swatch still matches its chart series

### Requirement: Statistics copy states how the figures are produced
The system SHALL label the artworks chart as stacked by school and by the artist's birth period, and SHALL label each chart table as that chart's data. The footnote SHALL state how often the figures are recomputed in a way the implementation keeps true.

#### Scenario: Visitor reads the Statistics copy
- **WHEN** a visitor opens the Statistics route
- **THEN** the artworks chart reads "STACKED BY SCHOOL · BIRTH PERIOD OF ARTIST", each table caption ends "— DATA", and the footnote reads "RECOMPUTED HOURLY FROM PUBLISHED RECORDS."
