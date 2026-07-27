# WGA Homepage Implementation Plan

## Introduction

Implement the public homepage from the approved WGA Home design while preserving
existing routes, server-rendered delivery, and the existing landing-page data.
The homepage uses dedicated WGA-owned styles that also support the shared
application components.

## Phase 1: Homepage delivery

### Item: Build the page

- [✓] Replace the homepage markup with semantic, responsive sections for the
  collection introduction, search, discovery links, collection figures, and
  existing welcome content. Verification: existing page routes and HTMX
  navigation attributes remain valid.
- [✓] Add scoped homepage styling that does not use daisyUI component classes.
  Verification: desktop and mobile layouts retain readable contrast, visible
  focus styles, and no dependency on daisyUI component styling.
- [✓] Regenerate Templ output and frontend assets. Verification: generated
  files are produced locally and remain untracked as required.

### Item: Verify and deliver

- [✓] Run focused template, frontend, and Go checks. Verification: each command
  succeeds with no new diagnostics.
- [ ] Manually review the homepage at desktop and mobile widths. Verification:
      the primary search and collection routes are visible and usable.
- [✓] Commit the implementation and open a pull request for #45. Verification:
  the pull request describes the homepage, Explore Artworks, and daisyUI
  removal work, and links to the parent issue.

## Phase 2: Explore Artworks delivery

### Item: Redesign collection search

- [✓] Rework the artwork-search page into the approved Explore Artworks layout.
  Verification: the search controls, filter sidebar, result header, and artwork
  grid remain responsive and use no daisyUI component classes.
- [✓] Preserve normal GET navigation and the existing HTMX result-fragment,
  pagination, and dual-mode contracts. Verification: filtering, clearing, and
  paginating results continue to update only the intended result region.
- [✓] Add focused browser coverage for the redesigned Explore Artworks page.
  Verification: the page loads, a search returns results, and the browser URL
  remains shareable.

## Phase 3: daisyUI removal

### Item: Replace shared UI primitives

- [✓] Replace daisyUI themes and component styles with WGA-owned CSS tokens and
  primitives. Verification: light and dark theme controls, navigation, forms,
  dialogs, alerts, toasts, cards, tables, and pagination retain their required
  behaviour without daisyUI.
- [✓] Remove daisyUI from the package and lockfiles. Verification: the frontend
  build no longer loads the daisyUI PostCSS plugin or package.
- [✓] Run application and browser regression checks. Verification: the public
  pages, feedback dialogs, toast notifications, theme switching, search, and
  Dual Mode continue to function.
