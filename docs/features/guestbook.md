# WGA Guestbook — Feature Spec & Acceptance Criteria

> Target: Modernize the legacy WGA guestbook experience currently served via `index_guest.html` / `guestbook.html`, while preserving archival content, submission flow, and nostalgic framing. Backend: PocketBase (entries, moderation state) + serverless function for mail/notifications. Frontend: HTMX-rendered views with progressive enhancement.

---

## 1) Scope Overview

**User-facing:**

- Guestbook landing accessible from main navigation.
- Overview intro encouraging visitors to leave a message.
- Year-based archive switcher (current year + historical tables).
- Paginated list of guest entries showing message, author, location, timestamp.
- Prominent CTA leading to the submission form.

**Admin/Content:**

- Moderation queue for newly submitted entries (approve, edit, reject).
- Tools to manage spam and abusive content (flag, ban email/IP).
- Ability to export entries (CSV/JSON) for preservation.

---

## 2) Information Architecture

1. **Landing shell**: replaces legacy frameset (`header` / `guestbook.html` / `panel`) with responsive layout.
2. **Hero intro**: short welcome paragraph and `Add to Guestbook` button.
3. **Archive bar**:
   - Primary nav row listing recent years (e.g., `2024`, `2023`, `2022`).
   - Secondary dropdown for older years (back to first recorded year).
4. **Entry list**:
   - Chronological (latest first) within selected year.
   - Each entry includes comment, author name (bold), optional email link, city/state/country, and formatted date.
   - Divider or subtle border between entries, matching accessible contrast.
5. **Submission form** (`/guestbook/new`):
   - Fields: Name (required), Email (optional, validated), City, State/Province, Country, Comment.
   - Honeypot/CAPTCHA or rate limiting to mitigate spam.
   - Terms snippet clarifying public display & consent.
6. **Success flow**:
   - Confirmation page with moderation notice (“Your entry awaits review”) or immediate publish message.
   - Link back to selected year, highlight pending entry if auto-approved.

---

## 3) Data Model (PocketBase)

### `guestbookEntries`

| Field | Type | Notes |
|-------|------|-------|
| `id` | text | UUID |
| `year` | number | Derived from `createdAt` |
| `name` | text | Required |
| `email` | email? | Optional; stored hashed for privacy |
| `city` | text? | Optional |
| `state` | text? | Optional |
| `country` | text? | Optional; ISO-3166 suggested |
| `comment` | text | Markdown or sanitized HTML |
| `source` | text | `web`, `admin`, `import` |
| `status` | text | `pending`, `approved`, `rejected`, `archived` |
| `moderatedBy` | relation? | Admin user reference |
| `submittedIp` | text? | Stored hashed/salted |
| `createdAt` / `updatedAt` | timestamp | Auto-managed |

**Indexes:** `index(year, createdAt desc)`, `index(status)`, `index(source)`.

---

## 4) Interaction & UX Flow

1. Visitor lands on `/guestbook` (current year view).
2. Archive switcher updates content via HTMX (`hx-get="/guestbook/2023"`).
3. “Add to Guestbook” triggers modal or navigates to `/guestbook/new`.
4. Form submission POSTs to backend; response indicates moderation outcome.
5. If approved instantly, page scrolls to anchor highlighting the new entry.
6. Admins access `/admin/guestbook` for moderation dashboard (filters, bulk actions).

---

## 5) Integration & Migration Notes

- Import legacy HTML archives (`/support/guest/guestbookYY.html`) into PocketBase.
- Preserve historical formatting (line breaks, italics) while sanitizing scripts/unsafe tags.
- Map legacy mailto links; redact or obfuscate emails to reduce scraping.
- Replace CGI endpoint (`/cgi-bin/guestbook.cgi`) with PocketBase action or cloud function.
- Trigger optional notification (email/Slack) when new entry awaits moderation.
- Provide RSS/JSON feed for new entries if useful for community updates.

---

## 6) Non-Functional Requirements

- **Accessibility**: WCAG 2.1 AA, proper heading hierarchy, focus management for modals, visible focus states.
- **Performance**: Archive load < 600 ms server time; lazy load older years (>200 entries) with pagination or infinite scroll.
- **Security**: Input sanitization, rate limiting (per IP/email), spam detection (Akismet or custom heuristics).
- **Privacy**: Email stored hashed; display as `mailto` only when contributor opts in; GDPR/consent statement.
- **Observability**: Log submissions, moderation actions, and spam rejections; metrics for daily submissions.

---

## 7) Acceptance Criteria

- [ ] `/guestbook` renders current year entries with intro, archive switcher, and consistent styling.
- [ ] Archive controls allow navigation to any prior year without full page reload.
- [ ] Entry cards display comment, author, optional contact/location, and submission date matching legacy content.
- [ ] Submission form validates required fields and blocks obvious spam (CAPTCHA or honeypot).
- [ ] Successful submission surfaces confirmation and queues entry for moderation (unless auto-approved rules apply).
- [ ] Admin dashboard lists pending entries with approve/reject actions, audit trail, and bulk controls.
- [ ] Imported legacy entries retain original ordering and timestamps within each year.
- [ ] Email addresses display only when authors consent and use obfuscation to deter scraping.
- [ ] Error states (validation, network, moderation rejection) provide actionable feedback.
- [ ] Regression suite covers archive navigation, submission flow, moderation actions, and legacy import scripts.

---

## 8) Open Questions & Follow-Ups

- Desired moderation SLA and whether auto-approval is acceptable for trusted domains?
- Should visitors be able to sort/filter entries (e.g., by country, keyword)?
- Do we expose an RSS feed or API endpoint for recent guestbook messages?
- Retention policy for IP/email hashes; any legal requirements for deletion upon request?
- Need for email verification step before publishing (double opt-in)?
- How prominently should cross-promotions (donations, postcards) appear in the redesigned layout?

