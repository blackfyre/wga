# Legacy Dual-Window Parity Specification

> Reference: <https://www.wga.hu/index_co_dual.html> and its linked Dual-Window resources, reviewed 19 July 2026.  
> Related current feature: [Dual Mode](dual-mode.md) at `/dual-mode`.
> Tracking issue: [#164](https://github.com/blackfyre/wga/issues/164).

## 1. Introduction

The legacy Dual-Window View lets visitors study two gallery pages at once: related works, companion pieces, different views of a sculpture, or an artist biography beside works. It also supports two-page printing.

This specification defines feature parity as preserving those user outcomes in the modern application. It does **not** require reproducing the legacy frameset, image-button interface, or its accessibility limitations.

### Current readiness

**Core comparison is ready; legacy feature parity is incomplete.** The current `/dual-mode` route already renders two independently scrollable panes, supports artist and artwork content, preserves pane state in the URL, and routes supported artist/artwork links to either pane. The remaining gaps are discovery, pane-management controls, legacy catalogue entry points, printing, and the explicit exit/music actions.

| Legacy capability                                                                      | Current status | Evidence                                                                                                 |
| -------------------------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------------------- |
| Two independently rendered and scrollable desktop panes                                | Complete       | `internal/assets/templ/pages/dual.templ`; `playwright-tests/dual-mode.spec.ts`                           |
| Compare artist, artwork, and embedded biography content                                | Complete       | `internal/handlers/dual/main.go` renders artist and artwork blocks                                       |
| Keep pane state in a reloadable/shareable URL                                          | Complete       | `left`, `right`, and `*_render_to` query parameters                                                      |
| Choose whether supported artist/artwork links replace a pane or open in the other pane | Complete       | Per-pane labelled toggle and Playwright coverage                                                         |
| Load content through a chooser                                                         | Partial        | A JavaScript-populated, searchable artist chooser is available; artworks require a pasted canonical path |
| Use the legacy A–Z, catalogue-section, Search, and Hints entry points in a pane        | Missing        | `parsePanePath` accepts artist and artwork detail paths only                                             |
| Create artist lists by school, period, time-line, profession, and sort order           | Missing        | The current artist listing only supports a name query and cannot be loaded into a pane                   |
| Copy, reverse, and clear panes                                                         | Missing        | The modern control bar has no equivalent actions                                                         |
| Explicit standard-view exit and music-selection action                                 | Missing        | No equivalent Dual Mode controls; music handler registration is disabled                                 |
| Print two selected pages together                                                      | Missing        | No Dual Mode print experience or acceptance coverage                                                     |

The legacy page recommends a 1024 × 768 display. The modern view activates at 768 px; it must be visually verified at the legacy reference size before calling the desktop experience complete.

## 2. Target Experience

### Content and navigation

- Render a left and a right pane on desktop, with independent scroll positions.
- Support artists, artworks, and biography content already embedded in artist pages.
- Let a visitor load an artist or artwork into either pane without knowing a canonical URL.
- Provide modern, pane-safe equivalents for the legacy A–Z artist index, artist lists, Early Christian/Medieval, Decorative Arts, Architecture, Search, and Hints entry points. Do not expose an entry point until its destination can render in Dual Mode.
- Preserve the full pane state and link-target choices in a shareable URL.
- Retain a usable default pane when a selected record is unavailable.

### Pane controls

- Let each pane's links open in the same or opposite pane.
- Provide **Copy left to right**, **Copy right to left**, **Reverse panes**, **Clear left**, and **Clear right** controls.
- Provide an explicit, keyboard-accessible Standard view action that returns to `/` and intentionally discards the Dual Mode layout state.
- Expose per-pane search or a dual-safe route from search results so a result can be loaded directly into the selected pane.
- Link to Music selection only when the music feature is available; otherwise omit the action rather than presenting a dead control.

### Printing and accessibility

- Provide an A4-landscape print layout that includes both selected panes without global navigation, pane controls, feedback UI, or modal chrome.
- Work at 1024 × 768 and higher; retain the current clear desktop-only state below the supported breakpoint.
- Use labelled buttons and form controls, visible keyboard focus, logical focus order, and accessible dialog labelling and focus restoration for content selection.
- Keep the modern implementation frame-free and progressively usable without hover-only controls.

## 3. Delivery Phases

### Phase 1 — Current comparison foundation

#### Item: Core two-pane behaviour

- [x] Render artist and artwork content in two independent desktop panes. **Verified by:** `playwright-tests/dual-mode.spec.ts` loads an artist and an artwork in opposite panes.
- [x] Persist pane paths and per-pane link targets in the Dual Mode URL. **Verified by:** the same-pane target Playwright journey reloads successfully.
- [x] Show a valid default state when a selected record is missing. **Verified by:** the missing-record Playwright journey.
- [x] Show an explicit unsupported state below 768 px. **Verified by:** the small-screen Playwright journey.

### Phase 2 — Pane management and discovery parity

#### Item: Direct selection and pane operations

- [ ] Add searchable artist and artwork selection for either pane. **Done when:** a visitor can select either content type without manually constructing a path.
- [ ] Add copy, reverse, and clear controls. **Done when:** each operation updates both rendered panes and the shareable URL without a full document navigation.
- [ ] Add an explicit Standard view action. **Done when:** it has a clear destination and is operable by keyboard and screen reader.
- [ ] Add focused Go and Playwright coverage for every pane operation and URL-state transition. **Done when:** success and empty-pane paths are covered.

#### Item: Catalogue entry points and search

- [ ] Make the modern equivalents of the legacy A–Z, artist-list, period, Decorative Arts, Architecture, Search, and Hints entry points loadable into a selected pane, or document an intentional product exclusion for each unavailable content type. **Done when:** no visible Dual Mode entry point leads to unsupported content.
- [ ] Audit whether school, period, time-line, profession, and sort data are available in the modern catalogue; implement each supported filter or document its exclusion. **Done when:** every legacy filter has a backed query or an explicit product decision, and supported filter results are shareable and usable from Dual Mode.
- [ ] Provide a dual-safe search route or per-pane search control. **Done when:** a result can be selected into the intended pane from a keyboard-accessible search flow.
- [ ] Add integration and browser coverage for discovery paths. **Done when:** one path from each supported catalogue entry point loads content into the selected pane.

### Phase 3 — Supporting parity outcomes

#### Item: Print, music, and accessibility

- [ ] Add a two-pane print stylesheet and print action. **Done when:** A4-landscape browser print preview contains both selected panes and excludes application chrome.
- [ ] Restore or intentionally exclude the Music selection action based on the music feature's delivery status. **Done when:** the Dual Mode UI contains no dead music link.
- [ ] Complete an accessibility pass for the controls and selection dialog. **Done when:** automated checks and keyboard-only manual testing cover focus, labels, dialog focus restoration, and control order.
- [ ] Perform a manual desktop review at 1024 × 768 and capture the result in the tracking issue. **Done when:** comparison, navigation, controls, and print behaviour remain usable at the legacy reference size.

## 4. Acceptance Criteria

- [ ] A visitor can independently load and compare two artists, two artworks, or an artist and an artwork.
- [ ] A visitor can use supported catalogue, Search, and Hints entry points to load content into either pane without pasting a path.
- [ ] Copy, reverse, and clear operations are available, accessible, and reflected in the URL.
- [ ] Link-target settings, pane state, and operations survive reload and sharing.
- [ ] Each legacy artist-list filter has a backed, reproducible result or an explicit documented exclusion; supported results can be used from Dual Mode.
- [ ] The UI includes an explicit standard-view exit and no unavailable music action.
- [ ] A4-landscape print preview produces two usable panes on one sheet; desktop layout is separately verified at 1024 × 768 or higher.
- [ ] Desktop, small-screen fallback, keyboard, and screen-reader behaviour have focused automated checks or a documented manual accessibility procedure.

## 5. Non-Goals

- Reintroducing HTML frames, image-only controls, hover-only interaction, or the legacy layout.
- Supporting an active two-pane interface below the desktop breakpoint.
- Treating every existing or future site route as Dual Mode content without an explicit compatibility decision.
- Rebuilding the music subsystem solely to display a Dual Mode link.

## 6. Evidence Sources

- Legacy overview: <https://www.wga.hu/support/dual/title2.html>
- Legacy controls and entry points: <https://www.wga.hu/support/dual/footer_artist.html>
- Legacy artist-list filters: <https://www.wga.hu/art/index_dual_left.html>
- Current implementation: `internal/handlers/dual/main.go`, `internal/assets/templ/pages/dual.templ`, and `resources/js/app.ts`
- Current browser coverage: `playwright-tests/dual-mode.spec.ts`
