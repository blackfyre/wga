## Why

The Claude Design changelog entry of 2 October 2026, "Keyboard bar is permanent; floating controls clear every bottom bar", makes the keyboard hint bar a permanent desktop surface and places FEEDBACK a fixed gap above the whole bottom stack. The repository still reveals the bar only after the first key press, so the bottom stack changes height mid-session, the bar is not part of the measured stack, and FEEDBACK and the toast stack sit a flat 16px above the measured stack at every width.

## What Changes

- Show the keyboard hint bar from page load on devices with a hovering, fine pointer, without waiting for keyboard use; keep it hidden on touch and coarse-pointer devices.
- Register the bar as the lowest item of the measured bottom stack, so the trays, the cookie notice, the content reservation, FEEDBACK, and toasts all clear its 30px.
- Place FEEDBACK and the toast stack one shared gap above the total bottom-stack height: 16px below 720px and 24px from 720px.
- Reserve the bar's 30px in the pre-measurement bottom-stack fallback on desktop pointers.
- Reserve the bottom stack after the shared footer rather than after the main content area, so a page's final content, including the footer, clears every docked surface.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `keyboard-navigation`: the hint bar is visible on desktop pointers before any key press.
- `public-page-experience`: the bottom stack includes the keyboard hint bar, and floating controls clear it by a width-dependent constant gap.

## Impact

- Shared layout and keyboard templates, the shared stylesheet, the keyboard script, and Playwright specs for the bottom stack, feedback control, and keyboard navigation. No handler, data, or configuration changes.

## Non-goals

- The cookie notice's own placement rule (16px above the stack beneath it), the bar's contents, and the tray heights are unchanged.
- The repository has no "Keyboard hints always" tweak and no bar slide-in animation, so there is nothing to remove for those items.
