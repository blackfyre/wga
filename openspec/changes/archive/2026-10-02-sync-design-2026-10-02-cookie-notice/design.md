# Design

## Context

Consent logic already lives in `resources/js/cookieconsent.ts` (Vanilla CookieConsent 3.1.0) and `resources/js/preference-consent.ts`. The library renders both surfaces into `#cc-main`, and `resources/css/style.pcss` themes them.

- **Notice:** uses the `box inline` layout, bottom left. Its actions are two library button groups. The first holds `ACCEPT ALL` and `DENY`; the second holds `PREFERENCES`. On narrow screens the stylesheet stacks every button in one column. From 45rem the buttons sit in one row.
- **Preferences interface:** uses the library's centred `box` layout.
- **Footer note:** the site's own Preferences panel (`PreferencesPanel` in `footer.templ`) is a right-hand `<dialog>`, and its storage note is static text.
- **Privacy policy:** a `static_pages` record. Its cookie section is the heading `<h2>Cookies and Web Beacons</h2>` followed by one paragraph, up to the next `<h2>`.

## Goals / Non-Goals

**Goals:**

- Match the design's notice hierarchy and wrapping, and its preferences panel placement and focus behaviour, using the library's own structure.
- Keep consent state, storage gating and the bottom-stack contract exactly as they are.

**Non-Goals:**

- Replacing Vanilla CookieConsent, or moving the preferences categories into the site's own Preferences dialog.
- Renaming storage keys to the prototype's `wga-board` and `wga-consent`.

## Decisions

### Use the library's `bar` layout for the preferences interface

Set `preferencesModal.layout` to `bar` with `position: right`. The library then renders the interface as a full-height sheet at the right edge. At or below the 640px breakpoint the stylesheet makes it cover the full viewport. Its width, border and header follow the site's Preferences panel.

- **Alternative:** render the categories inside the site's own `<dialog>` and drive the library through its API.
  - **Why rejected:** it duplicates the library's category state, toggles and save flow. Two consent UIs would then have to agree.

### Wrap the notice's existing button groups instead of reordering them

Keep the library's two button groups.

- Make `.cm__btns` a wrapping row.
- The first group holds `ACCEPT ALL` and `DENY` as two equal flex items.
- The second group takes the full width whenever the first group fills the row. On a wide notice all three fit on one line.

This removes the stylesheet's column-at-narrow rule and the 45rem switch, which together are the "switch layout at a breakpoint" behaviour the design replaces.

The library's `box inline` layout already places the text above the actions at every width, so the layout option stays; only the button rules change. The library's narrow-screen rules stack the groups in columns with `!important`, so the override restates the row direction under the same 640px query.

### Measure the notice after the library reveals it

The library inserts the notice hidden and reveals it about a frame later by toggling classes on `<html>` and on its wrapper. Neither the DOM tree nor the notice's size changes at that moment, so the bottom-stack observers never measured it after the reveal. With the keyboard bar now permanent on desktop (#256), the notice then sat at its 16px fallback, on top of the bar. The bottom-stack observer now also watches `class` and `hidden` attribute changes across the document. Measuring writes only style properties, so the extra observation cannot feed back into itself; the idle-write test guards that.

### Focus, inertness and the returning notice

- **Inertness:** rely on the preferences interface's own overlay and focus trap. They block pointer input to the page behind it and keep focus inside. Leave `disablePageInteraction` unset: that option blocks the page while the notice is shown, and the design does not ask for it.
- **Focus on open:** an `onModalShow` handler for `preferencesModal` moves focus to the library's close button. That button is labelled `CLOSE` visibly, not only through `closeIconLabel`.
- **Notice returns:** if closing the interface before any choice does not restore the notice by itself, an `onModalHide` handler calls `CookieConsent.show()` while `CookieConsent.validConsent()` is false.

All three are checked in Playwright instead of assumed from library defaults.

### Drive the footer note from consent state

The consent decision handler already runs on every grant, withdrawal and cross-tab change. It sets `data-wga-preference-storage="on|off"` on `<html>`. The panel renders both note variants, and CSS shows the one that matches.

Without JavaScript the attribute is absent, and the visit-only note shows. That is correct, because consent cannot be granted without the script.

### Rewrite only the privacy policy's cookie section, in a migration

The migration replaces the `<h2>Cookies and Web Beacons</h2>` heading and its content up to the next `<h2>` with the new section. Everything else is left untouched.

When the heading is absent, the migration logs that it skipped the record and leaves it unchanged. This covers an already-edited production page or a reseeded fixture. Rollback returns an error, as `seed_about_page` does, because the replaced text was not preserved.

## Risks / Trade-offs

- **[Library markup or classes change on upgrade]** → The version stays pinned to 3.1.0. The Playwright geometry checks (row membership, 44px heights, panel edge) fail visibly on a Dependabot bump.
- **[Production privacy content differs from the fixture]** → The migration matches on the section heading and does nothing when it is missing, so it never overwrites unknown content. After deploy, check that the live page shows the new section.
- **[The library's overlay does not make the page inert for assistive technology]** → The Playwright check confirms that focus cannot leave the panel. If the panel's `aria-modal` semantics prove insufficient, set `inert` on the page content while it is open.

## Migration Plan

Deploying applies the privacy migration on server start. Rollback of the code is a normal revert. The privacy text change is forward-only.
