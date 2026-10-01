## Context

The recipient handler looked up the artwork record and shaped the page itself. The artwork record page derives dimensions and the holding location from the record's comment summary and its current-location relation. Several packages repeat that parsing.

## Decisions

- **The postcard workflow owns the received-card projection.** `internal/postcards` resolves the published artwork, its complete artist identity, the canonical record URL, the compose URL and the record details. The handler resolves the token, maps the projection to the page and records receipt.
- **Record details come from the artwork package.** The location and dimensions parsing and the current-location lookup move to `internal/artworks`, which both the artwork record handler and the postcard workflow use. The other copies (Dual Mode, study board, agent content) stay as they are; consolidating them is outside this change.
- **The record link names a published author.** The card uses the work's first published author with complete identity, because the artwork route serves only published artists. With no such author, the page returns 404, as it already did for incomplete identity.
- **The summary may omit dimensions.** The importer writes `date · location` when the source has no dimensions, so a two-part summary yields the location and no dimensions.
- **The holding location prefers the current-location record.** It falls back to the location in the comment summary, as the study board does, and is omitted when neither exists.
- **Card links are plain anchors.** The page is served with `Referrer-Policy: no-referrer`, so a full navigation does not disclose the bearer URL to the destination. A full navigation also leaves the token URL, whereas an un-pushed `hx-get` would swap content but leave the address bar on that URL. `BROWSE THE ARCHIVE →` changes from `hx-get` to a plain anchor for the same reason.
- **Shared layout navigation keeps HTMX.** Its links send `HX-Current-URL` only to the WGA origin that issued the token, and the application does not log or forward that header. Disabling HTMX for the whole recipient document would change shared shell behaviour, which is outside this change.

## Risks

- A record whose comment summary does not follow the `date · location · dimensions` shape yields no dimensions. The line is then omitted rather than guessed.
