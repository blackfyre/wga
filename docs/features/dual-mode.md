# WGA Dual-Mode Architecture Spec (Public + Admin)

> Extension of: WGA Artist Directory Spec  
> Backend: PocketBase  
> Frontend: HTMX  
> Modes:  
> • Public — read-only browsing  
> • Admin — authenticated editorial mode  

---

## 1. Modes Overview

| Mode | Audience | Access | Purpose |
|------|-----------|---------|----------|
| **Public** | Visitors | Anonymous | Browse artists, artworks, and metadata |
| **Admin** | Curators | Authenticated (PocketBase Auth) | Manage, import, and curate data |

Both modes share the same codebase but use distinct routes and permission rules.

---

## 2. Mode Switching

### Requirements
- Shared PocketBase backend; routes split logically:
  - `/` → Public browsing
  - `/admin` → Admin dashboard
- Authentication: PocketBase Auth (email/password or OAuth)
- Role-based visibility for UI elements
- Persistent session tokens scoped to `/admin`
- Graceful fallback to login on timeout

### Acceptance Criteria
- [ ] `/admin` redirects to login if not authenticated
- [ ] Public users never see edit/import controls
- [ ] Switching between modes clears state correctly
- [ ] Sessions expire and redirect to login without errors

---

## 3. Shared Components

### Common Layout
- Header, footer, breadcrumbs shared across both modes
- Unified design system (responsive grid, dark/light)
- Notification system (success, warning, error)

### Data API
- Shared PocketBase API endpoints
- Public: read-only (GET)
- Admin: CRUD via authenticated requests

### Acceptance
- [ ] Templates consistent between modes
- [ ] Data endpoints identical except access control

---

## 4. Admin Dashboard Structure

| Route | Description |
|--------|--------------|
| `/admin` | Overview of key metrics and recent imports |
| `/admin/artists` | Artist table with CRUD |
| `/admin/artworks` | Artwork table and image upload |
| `/admin/imports` | Import manager (upload + logs) |
| `/admin/duplicates` | Duplicate review and merge interface |
| `/admin/images` | Image integrity dashboard |

---

## 5. Admin UI Components (HTMX)

### Artist Table
- Server-rendered table with sortable columns
- Modal for details and edit
- Tabs for Bio, Works, Metadata

### Artwork Table
- Filters by artist, year, medium
- Inline image preview and upload

### Import Manager
- Upload form with progress polling (`hx-trigger="every 5s"`)
- Log area auto-updates

### Duplicate Review
- Diff panel comparing two records
- Merge and dismiss buttons trigger PB functions

### Acceptance Criteria
- [ ] CRUD modals submit via HTMX and refresh in place
- [ ] Imports auto-refresh while running
- [ ] Merges create consolidated record and archive dupes

---

## 6. Public vs Admin Feature Matrix

| Feature | Public | Admin |
|----------|---------|-------|
| A–Z navigation | ✅ | ✅ |
| Artist listings | ✅ | ✅ |
| CRUD actions | ❌ | ✅ |
| Image upload | ❌ | ✅ |
| Import jobs | ❌ | ✅ |
| Duplicate handling | ❌ | ✅ |
| Search | ✅ | ✅ |
| Caching | ✅ (1h TTL) | ❌ |
| Logging | minimal | verbose |

---

## 7. Backend Access Rules (PocketBase)

### Example Rule: `artists`
```json
{
  "listRule": "true",
  "viewRule": "true",
  "createRule": "@request.auth.role='admin'",
  "updateRule": "@request.auth.role='admin'",
  "deleteRule": "@request.auth.role='admin'"
}
```

### Acceptance
- [ ] Admin API token grants CRUD access
- [ ] Public requests restricted to GET
- [ ] Access failures logged with user and timestamp

---

## 8. Deployment Strategy

- Single PocketBase + HTMX instance  
- Reverse proxy: `/admin` requires auth, `/api` public cached  
- Optional: separate subdomains (e.g., `wga.hu` vs `edit.wga.hu`)

### Acceptance
- [ ] Cache headers applied for public endpoints
- [ ] Admin served via authenticated route
- [ ] Admin pages excluded from sitemap

---

## 9. Dual-Mode Testing

| Area | Type | Validation |
|------|------|-------------|
| Auth | Unit | Login/logout works |
| Routing | Integration | `/admin` locked; `/artists` open |
| CRUD | E2E | Admin edits succeed; public blocked |
| Imports | Integration | Upload + polling OK |
| Duplicates | Unit | Merge/dismiss logic correct |
| Cache | Functional | Public cache invalidates after edits |

---

## 10. Definition of Done

- [ ] Full admin CRUD via web UI
- [ ] Public pages unaffected by admin actions
- [ ] Cache headers active and verifiable
- [ ] Access control tested for each collection
- [ ] End-to-end import/edit cycle passes QA

---

## 11. Optional Add-ons

- Editor role with partial write rights
- Audit trail collection (`changes`)
- Activity feed (`recent edits`, `imports`)
- Diff-based change viewer
- WebSocket progress updates for imports

---
