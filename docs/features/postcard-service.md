# WGA Postcard Service — Feature Spec & Acceptance Criteria

> Target: Rebuild the beloved postcard workflow (currently under `index_post.html` → `support/post/sendcard.html` → CGI pipeline) with a modern stack while honoring legacy behavior and art selections. Backend: PocketBase for postcard templates and message queue, paired with transactional email (e.g., Resend, Postmark). Frontend: HTMX-enhanced pages, no dependency on legacy frameset.

---

## 1) Scope Overview

**User-facing:**

- Postcard landing page explaining the service and surfacing curated art categories.
- Selection experience for choosing artwork (preview grid + metadata).
- Postcard composition form (sender name/email, recipient email(s), message, optional music track).
- Confirmation screen and success email to recipient with chosen artwork and optional pickup link.
- Optional postcard archive URL (view-only page for recipients).

**Admin/Content:**

- Manage postcard templates (artwork image, artist attribution, description).
- Curate category groupings (General, Christian, Special Themes, seasonal collections).
- Configure default background, typography, and optional accompanying audio tracks.
- Monitor delivery status (queued, sent, bounced) with retry controls.

---

## 2) Information Architecture

1. **Landing (`/postcards`)**
   - Welcome text mirroring legacy introduction.
   - CTA button “Create a postcard”.
   - Category cards (thumbnail + description) linking to filtered selection views.
   - Cross-promo block highlighting Fine Arts in Hungary partner site.
2. **Category Selection (`/postcards/category/<slug>`)**
   - Responsive grid of artwork cards (thumbnail, title, artist).
   - Filters: period, medium, popular, seasonal.
   - Search bar to find artwork by keyword or inventory ID.
3. **Artwork Detail (`/postcards/art/<id>`)**
   - Larger preview, metadata, link to original artwork page.
   - “Use this artwork” action launching composition form.
4. **Composer (`/postcards/new?art=<id>`)**
   - Form fields:
     - Sender name (required)
     - Sender email (required; used for receipts)
     - Recipient email(s) (required; allow multiple validated addresses)
     - Optional greeting subject
     - Personalized message (textarea with character count)
     - Optional music track selection (reuse music library)
     - Consent checkbox (agree to terms/privacy + permission to email recipients)
   - Anti-spam controls (reCAPTCHA, rate limiting, link cool-down).
5. **Confirmation**
   - Show postcard preview (image, message, audio selection).
   - Delivery status indicator (sent immediately vs queued).
   - Link to create another postcard.
6. **Recipient Experience**
   - HTML email using responsive template (includes image, message, audio link).
   - Mirror page hosted at `/postcards/view/<token>` for web playback and compliance with email clients.

---

## 3) Data Model (PocketBase)

### `postcardTemplates`

| Field | Type | Notes |
|-------|------|-------|
| `id` | text | UUID |
| `artworkRef` | relation? | Link to artworks collection |
| `title` | text | Display name |
| `artist` | text | Denormalized for email |
| `description` | text? | Optional |
| `image` | file | Primary postcard image |
| `categories` | text[] | e.g., `general`, `christian`, `seasonal` |
| `tags` | text[] | Search keywords |
| `isActive` | bool | Toggle availability |

### `postcards`

| Field | Type | Notes |
|-------|------|-------|
| `id` | text | UUID |
| `template` | relation | Reference to `postcardTemplates` |
| `senderName` | text | Required |
| `senderEmail` | email | Required |
| `recipientEmails` | json[] | Validated list |
| `subject` | text? | Optional override |
| `message` | text | Markdown or sanitized HTML |
| `musicTrack` | text? | URL or track ID |
| `status` | text | `queued`, `sent`, `delivered`, `bounced`, `failed` |
| `deliveryMeta` | json? | Provider response |
| `expiresAt` | timestamp? | For view link expiry |
| `createdAt` / `updatedAt` | timestamp | Auto-managed |

---

## 4) Interaction & UX Flow

1. Visitor enters postcard flow (landing or direct category).
2. Chooses artwork; selection persists in session/local storage.
3. Completes composer form with recipient details and optional music.
4. Form submits via HTMX/Fetch to backend endpoint:
   - Validates inputs, stores record with `queued` status.
   - Enqueues email job (PocketBase `@request.auth` or external worker).
5. Worker composes email (HTML + plaintext) with signed image URL and optional audio.
6. Recipient receives email; link opens web view to access postcard and play music.
7. Sender optionally gets confirmation email with status + resend link.
8. Admin dashboard shows usage metrics and delivery outcomes.

---

## 5) Integration & Migration Notes

- Import existing curated lists (`support/post/*.html`) into `postcardTemplates` with categories.
- Map legacy `postcard.cgi` parameters to new API contract (image path, music path).
- Maintain compatibility with existing artwork pages by updating “Send as postcard” buttons to new routes.
- Provide redirect from legacy CGI endpoint to new composer (with `image` param -> template).
- Update analytics (GA4) to track postcard creation and delivery events.
- Ensure email provider configured with SPF/DKIM; include unsubscribe/manage preferences link.

---

## 6) Non-Functional Requirements

- **Accessibility**: All forms labeled, color contrast for previews, keyboard-friendly selection grid, accessible email fallback.
- **Performance**: Category pages load < 1s (pre-render views, cache thumbnails); composer loads music options lazily.
- **Security**: Validate all inputs, escape user-generated message, rate limit by IP/email, block open mail relays.
- **Compliance**: GDPR consent, ability to delete postcard data on request, respect privacy laws for marketing.
- **Reliability**: Email retries with exponential backoff; fallback to static link if music streaming fails.
- **Observability**: Audit log for postcard creation and delivery; metrics for daily volume, failure rate.

---

## 7) Acceptance Criteria

- [ ] `/postcards` landing loads without frames, offers category navigation, and surfaces partner link.
- [ ] Category grid displays curated templates, respects active flag, and supports search/filter.
- [ ] Composer form pre-fills selected artwork, validates inputs, and prevents spam submissions.
- [ ] Music selection dropdown matches legacy catalog and can be filtered/searchable.
- [ ] On submit, postcard record stored with metadata and delivery job enqueued; sender sees confirmation.
- [ ] Emails render the selected artwork, message, sender attribution, and optional music link consistently across major clients.
- [ ] Recipient view page reflects postcard content and honors expiration/consent rules.
- [ ] Legacy “Send as postcard” buttons redirect seamlessly to new composer.
- [ ] Admin dashboard lists postcard activity, delivery status, and supports resends/cancellations.
- [ ] Monitoring alerts on sustained delivery failures or spike in rejection/bounce rate.

---

## 8) Open Questions & Follow-Ups

- Do we allow multiple recipients per postcard? If so, how to handle personalization?
- Should senders be able to schedule delivery for future dates?
- Are there size limits or moderation requirements for user messages?
- Is music optional or required? Do we trim legacy list to cleared tracks only?
- Desired retention window for postcard data (auto-purge after X days/weeks)?
- Should postcards be publicly shareable (e.g., unique link) or private to recipients only?
