# Proposal

## Why

The Claude Design changelog entry of 2 October 2026, "Cookie notice: three-button consent, and preference storage needs it", sets out the presentation of the consent notice and of the COOKIE PREFERENCES panel. Most of its consent behaviour already shipped with `gate-preference-storage-behind-consent` (#252). The presentation still differs from the design:

- The notice stacks its three buttons in a column on narrow screens.
- COOKIE PREFERENCES opens as a centred box. The design uses the site's right-hand panel, which becomes a full-screen sheet on mobile.
- The footer note in the Preferences panel gives the same text whatever the consent state.
- The privacy page's cookie section is generic boilerplate. It says cookies store visitors' preferences, which contradicts the consent model.

## What Changes

- **Notice:** text and buttons stack at every width. ACCEPT ALL is the primary button, filled with the accent colour. DENY and PREFERENCES are outlined secondary buttons of the same size. All three have a 44px minimum height. The buttons wrap instead of switching layout at a breakpoint:
  - on desktop all three sit in one row;
  - on a narrow screen ACCEPT ALL and DENY share the first row, and PREFERENCES takes the full width beneath.
- **COOKIE PREFERENCES panel:** it opens from the notice's PREFERENCES button and from the footer's "Cookie settings". It is a right-hand panel on desktop and a full-screen sheet on mobile. Its buttons follow the notice's layout, and closing it without choosing brings the notice back. It has the Preferences panel's focus behaviour:
  - the rest of the page is inert;
  - focus moves to CLOSE;
  - Esc closes it.
- **Preferences panel footer note:** while preference storage is off, the note says that choices last for this visit only. While it is on, the note says they are kept on this device.
- **Privacy page:** a migration replaces the "Cookies and Web Beacons" section of the `privacy-policy` static page. The new section says that essential cookies cover the session and the consent record, that reading preferences are optional storage kept on this device with consent, and that no advertising or cross-site tracking cookies are set. It does not mention period music.
- **Design sync record:** `docs/design-sync.md` records the entry as implemented.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `public-page-experience`: the cookie-consent requirement gains its presentation: the button hierarchy and wrapping, the preferences panel's placement and focus behaviour, and the notice's return after the panel closes without a choice. The footer preferences requirement gains the consent-dependent storage note. The privacy page's cookie statement becomes a requirement.

## Impact

- `resources/js/cookieconsent.ts`: notice and preferences layout options, panel close behaviour.
- `resources/css/style.pcss`: the `#cc-main` notice and panel rules.
- `internal/assets/templ/components/footer.templ`: the Preferences panel's storage note.
- `resources/js/preference-consent.ts`, or the module that owns the note: switching the note on consent changes.
- `internal/migrations/`: a new migration that rewrites the privacy page's cookie section, with a test.
- `playwright-tests/`: the cookie-consent, bottom-stack and preference-storage specs.
- Non-goals:
  - **Storage key names unchanged.** The design's key names are `wga-board` and `wga-consent`. The repository keeps `wga-study-board`, and keeps the library's `cc_cookie` as the consent record. The specified behaviour is unchanged.
  - **Notice placement unchanged:** its position and bottom-stack offset stay as they are.
  - **Mobile FEEDBACK rule unchanged:** FEEDBACK stays hidden on mobile while the notice is shown.
