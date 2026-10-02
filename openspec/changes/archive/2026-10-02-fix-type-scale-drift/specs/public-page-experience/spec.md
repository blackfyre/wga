## ADDED Requirements

### Requirement: Rich-text prose and links follow the design type scale

The system SHALL render running prose in sanitised rich text (artist biographies, artwork and selection commentaries, Dual Mode panes, static pages, guided tours, and postcards) at `--t-16` with a 1.7 line-height in the text colour role, unless the surface explicitly selects another rung of the type scale, in which case that rung SHALL apply. Links inside that prose SHALL use the accent colour role with a 1px underline offset clear of the descenders, SHALL NOT change the surrounding weight, and SHALL take the secondary accent on hover. Headings, lists, code, quotations and strong text inside rich text SHALL use rungs of the type scale and the palette's colour roles. Public templates and styles SHALL NOT use a type size outside the 33-rung scale, including Tailwind's named `text-lg`, `text-xl` and `text-2xl` sizes. The keyboard focus ring SHALL use the active palette's accent colour role in every palette and scheme.

#### Scenario: Visitor reads an artist biography, artwork commentary or static page

- **WHEN** a visitor opens an artist biography, an artwork commentary, or a static page at 390px, 834px or 1440px in the light or dark scheme at a 16px root
- **THEN** each prose paragraph renders at 17px with a 28.9px line-height in the text colour role, and each prose link renders in the accent colour with a 1px underline at the paragraph's weight.

#### Scenario: Surface selects a different prose rung

- **WHEN** a prose surface such as a Dual Mode pane declares `--t-15`
- **THEN** its paragraphs render at that rung rather than the default prose rung.

#### Scenario: Visitor focuses a control in a dark palette

- **WHEN** a visitor moves keyboard focus in any palette and scheme
- **THEN** the focus ring uses that palette's accent colour role.
