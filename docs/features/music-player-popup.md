# WGA Music Player Popup — Implementation Notes & Acceptance Criteria

> Target: Recreate the legacy WGA classical music experience that launches from the on-site "music player" trigger, preserves existing catalog depth, and upgrades cross-device reliability. Backend: reuse current static assets and/or PocketBase for metadata. Frontend: progressive enhancement with vanilla HTML/CSS/JS (no framework runtime requirement), compatible with the broader HTMX-driven stack.

---

## 1) Scope Overview

**User-facing:**

- In-page "Listen to music" trigger accessible from desktop and mobile.
- Legacy-style track selector (`music2.html`) showing curated categories and track lists.
- Dedicated popup/overlay player that loads the selected track and exposes playback controls.
- Graceful fallback when popups are blocked (open in same tab or inline modal).
- Track metadata displayed alongside playback (title, composer, length, optional artwork).

**Admin/Content:**

- Maintainable catalog of tracks grouped by category (e.g., Baroque, Renaissance, Instrumental).
- Ability to add, edit, or retire tracks without redeploying the entire site.
- Optional analytics hooks to gauge feature usage.

---

## 2) Interaction & UX Flow

1. Visitor clicks the music trigger (button/link in header or sidebar).
2. Track selector loads (popup window or modal overlay) displaying:
   - Category navigation (tabs or list).
   - Track list per category with play action.
   - Clear indication of currently playing selection.
3. Selecting a track launches/updates the music player view:
   - Autoplays the chosen audio if permitted by browser policies.
   - Provides standard controls (play/pause, seek, volume).
   - Offers "Next"/"Previous" navigation within the current category.
4. Player persists while the visitor browses the main site (popup remains, overlay stays pinned).
5. Closing the player stops playback and releases resources.

---

## 3) Functional Requirements

- Trigger can be activated via keyboard (focusable element with `Enter`/`Space` support).
- Track selector lists at least all legacy tracks with searchable/filterable interface.
- Audio files stream from existing CDN/static storage; support standard formats (MP3/OGG).
- Player reflects network/loading states (spinner, error messaging).
- Respect browser autoplay restrictions (prompt user interaction when needed).
- Track URLs are shareable/deep-linkable (e.g., `?track=bach_air`).
- Popup window size adheres to responsive breakpoints (mobile-friendly overlay fallback).
- Multi-language support ready (strings externalized for future translations).

---

## 4) Non-Functional Requirements

- Accessibility: WCAG 2.1 AA, including focus management when opening/closing popup.
- Performance: track selector loads under 1s on broadband; audio starts within 2s average.
- Reliability: recover gracefully from broken audio URLs (retry or show error).
- Compatibility: evergreen desktop browsers + Safari iOS + Chrome/Firefox Android.
- Observability: client-side event logging for play, pause, error (optional endpoint).

---

## 5) Acceptance Criteria

- [ ] Music trigger appears across target templates and is keyboard + screen reader accessible.
- [ ] Opening the trigger reveals the track selector with categorized tracks mirroring `music2.html`.
- [ ] Selecting a track opens or updates the player, autoplays after user gesture, and shows metadata.
- [ ] Playback controls (play/pause, seek, volume, close) function consistently across supported browsers.
- [ ] Popup and overlay fallbacks both keep playback alive while the user navigates the main site.
- [ ] Direct link with `?track=<id>` opens the selector/player with that track preselected.
- [ ] Blocked popup scenario presents inline modal fallback instead of failing silently.
- [ ] Error states for missing/unloadable audio display user-friendly messaging and allow retry.
- [ ] Basic analytics hook (e.g., `data-event="music_play"`) fires on play for future instrumentation.

---

## 6) Open Questions & Follow-Ups

- Source of truth for audio metadata—static JSON, PocketBase collection, or external CMS?
- Licensing confirmation for streaming existing catalog (verify rights before relaunch).
- Should playback persist across page reloads via Service Worker or `localStorage` state?
- Need for playlist randomization or "Play All" mode?
- Requirements for offline or low-bandwidth support?

