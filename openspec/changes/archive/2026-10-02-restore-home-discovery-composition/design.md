## Context

Source: the 1 October 2026 design parity audit, items HO-1 to HO-6, and the Claude Design project "WGA" home screen as read on 2 October 2026 (see `docs/design-sync.md`). The design's hero has two actions, BROWSE ARTISTS → and COMPARE TWO WORKS →; its stat strip reads REPRODUCTIONS, ARTISTS, SCHOOLS and CENTURY; and its three feature cards are TWO WINDOWS, TIMELINE and POSTCARD SERVICE, each with a right-aligned mono 11px action link in the action ink.

In the repository, `internal/assets/templ/pages/home.templ` renders three hero actions (artists, artworks, inspiration), the labels ARTWORKS, ARTISTS, SCHOOLS and PERIOD, and the feature cards SIDE BY SIDE, POSTCARD SERVICE and COMMUNITY. No card links to the Timeline.

The real routes are `/dual-mode` (`internal/handlers/dual`), `/timeline` (`internal/handlers/timeline`), `/postcard` (the tokenless postcard landing in `internal/handlers/postcards`, which is the public compose entry point; `/postcard/send` needs a selected artwork), and `/contributors`.

## Decisions

### The hero follows the design's two actions

BROWSE ARTISTS → stays primary and links to `/artists`; COMPARE TWO WORKS → is the secondary action and links to `/dual-mode`. BROWSE ARTWORKS → and FIND INSPIRATION → are removed from the hero. No route is lost: artworks remain one step away through the recent-additions ALL WORKS → link and the primary navigation, and Inspiration remains a primary navigation destination (`collection-discovery`, "Collection discovery presents the release information architecture"), which is rendered on the home page.

### The feature cards follow the design, and contributors stay reachable

The cards are TWO WINDOWS (OPEN DUAL MODE → to `/dual-mode`), TIMELINE (OPEN THE TIMELINE → to `/timeline`), and POSTCARD SERVICE (SEND A POSTCARD → to `/postcard`). The COMMUNITY card, "Help sustain the archive", is removed, as in the design. Its only route, `/contributors`, moves into the POSTCARD SERVICE card as MEET THE CONTRIBUTORS →, 10px below SEND A POSTCARD →, and also remains in the footer and the mobile navigation.

The card label, title, body and link sizes follow the design's inline styles: the label is the shared `wga-section-label`, the title is 22px semibold (`--t-22`, overriding `wga-section-title`'s 20px locally so other pages are unaffected), the body is 15px with 1.6 leading, and the links are mono `--t-11`, 1px letter-spacing, `wga-action-link` (the action ink), right-aligned through `wga-trailing-action`.

### "Period music optional" is not adopted

The design's postcard body ends "Period music optional." That sentence is stale in the design itself: on 14 September 2026 the design removed the sender's music opt-in, and the `postcard-sharing` specification says the composer SHALL NOT offer a sender-controlled music setting and that the received page shows any applicable period music derived from the work. The card body therefore reads "Free, no account, no address kept after delivery." A replacement sentence about period music was considered and rejected because it would add copy the design does not have.

### The repository keeps its factual corrections

The kicker reads "00 — PROVIDING EXPERIENCE SINCE 1996" (the design still says 1992), and the period value reads "3rd–early 20th" (the design still says 3rd–19th). The value adopts the design's lowercase setting, with the design's CENTURY label beneath it, so the strip reads as the design does while stating the corrected range. The design should follow the repository on both facts.

### Count labels and live counts

The labels become REPRODUCTIONS, ARTISTS, SCHOOLS and CENTURY in the design's order. The values stay the handler's live counts; the design's 52,800 and 4,300 are sample figures.

### Work-of-the-day caption

The design captions the plate with the title and then "VERMEER, c. 1665 →". The repository shows the artist and date as "Artist · date" without an arrow. The caption now sets the artist in capitals and ends with an arrow that is hidden from assistive technology. The date stays separated by " · " rather than the design's comma, because the repository's artist value is the filing name ("VERMEER, Johannes"), which already contains a comma. The whole card remains the link, as before, so the plate, title and caption share one target.

## Risks / Trade-offs

- Removing two hero actions lengthens the path to Inspiration by one navigation step. The primary navigation keeps it one click away, and the specification scenario is updated so that the home requirement names the design's routes.
- `collection-discovery`'s current scenario requires the home page to reach "guided discovery" directly. The modified requirement replaces that with the design's routes and states that Inspiration and Guided Tours are reached through the primary navigation. This is a deliberate specification change, not an implementation detail.
