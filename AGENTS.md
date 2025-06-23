```markdown
# AGENTS.md - Guidelines for AI Agents

This document provides guidelines and information for AI agents working with the Web Gallery of Art (WGoA) codebase.

## Project Overview

The Web Gallery of Art is a web application aimed at providing a modern, responsive, and user-friendly experience for browsing a collection of paintings, sculptures, and other art forms. It is a reimplementation of the original WGoA website, built with modern technologies.

## Key Technologies

*   **Backend:**
    *   **Go:** Version 1.23+ (see `go.mod`)
    *   **PocketBase:** Used as the primary backend framework, providing database, ORM, authentication, and admin UI.
*   **Frontend:**
    *   **Templ:** A Go templating language for generating HTML. Server-side rendering is the primary approach.
    *   **HTMX:** Used to enhance HTML with AJAX capabilities, allowing for dynamic updates without full page reloads.
    *   **TailwindCSS:** A utility-first CSS framework for styling.
    *   **DaisyUI:** A component library for TailwindCSS.
    *   **Bun:** Used for managing frontend dependencies and running build scripts for CSS and JS.
*   **Build & Deployment:**
    *   **Goreleaser:** Used for building and releasing Go applications.
*   **Testing:**
    *   **Playwright:** Used for end-to-end testing. Tests are located in the `playwright-tests/` directory.

## Getting Started & Development Workflow

### Prerequisites

1.  **Go:** Version 1.23 or later.
2.  **Bun:** Version 1.1 or later.
3.  **Templ:** `go install github.com/a-h/templ/cmd/templ@latest`
4.  **Environment Variables:** Create a `.env` file based on `.env.example` and populate it with the necessary credentials and configurations (S3, SMTP, etc.).

### Building the Application

*   **Full Build (Go & Frontend):**
    *   Run the `build.sh` script: `./build.sh`
    *   Alternatively, manually:
        1.  Generate Templ components: `templ generate`
        2.  Build Go binary: `go build -o wga`
*   **Frontend Development:**
    *   Install dependencies: `bun install`
    *   Run dev server (watches Templ and PostCSS files): `bun run dev`
    *   Build JS assets: `bun run build:js`

### Running the Application

1.  Ensure you have a configured `.env` file.
2.  Start the server: `./wga serve`
3.  The application will be accessible at `http://localhost:8090` by default.

### Database Migrations

*   Migrations are handled by PocketBase.
*   Migration files are located in the `migrations/` directory.
*   PocketBase applies pending migrations automatically on startup.
*   For development, `Automigrate` is set to `false` in `main.go`, meaning collection changes in the Admin UI won't automatically create migration files. You might need to create them manually or adjust this setting if needed.

## Codebase Structure & Conventions

### Go (Backend)

*   **`main.go`:** Entry point of the application. Initializes PocketBase, registers handlers, hooks, and cron jobs.
*   **`handlers/`:** Contains HTTP request handlers. Handlers typically interact with PocketBase services (DAO for database access, etc.) and use Templ for rendering HTML responses.
*   **`models/`:** Defines data structures (structs) that map to PocketBase collections. These models often embed `pocketbase/models.BaseModel`.
*   **`assets/templ/`:** Contains all Templ files.
    *   **`assets/templ/pages/`:** Templ components for complete pages.
    *   **`assets/templ/components/`:** Reusable UI components (e.g., navigation, footer).
    *   **`assets/templ/layouts/`:** Base layouts for pages.
*   **`utils/`:** Utility functions used across the application.
*   **PocketBase Integration:**
    *   Leverage PocketBase's `app.Dao()` for database operations.
    *   Use PocketBase's event hooks (`app.On...`) for extending core functionalities (see `hooks/` and `main.go`).
*   **Error Handling:** Standard Go error handling. For HTTP handlers, use `echo` context methods for responses (e.g., `c.JSON()`, `c.String()`, or rendering a Templ error page).
*   **Logging:** Use `app.Logger()` for logging within the PocketBase context.

### Templ (HTML Templating)

*   Follow the existing structure for organizing components and pages.
*   Use `templ generate` to compile `.templ` files into Go code. This is usually part of the `build.sh` script or `bun run dev`.
*   Pass data to templates as typed Go structs for type safety.

### HTMX

*   Attributes like `hx-get`, `hx-post`, `hx-swap`, `hx-target` are used directly in Templ components to define dynamic interactions.
*   Backend handlers should be designed to return HTML fragments that HTMX can swap into the page.

### CSS (TailwindCSS & DaisyUI)

*   Styles are primarily applied using TailwindCSS utility classes directly in the Templ files.
*   DaisyUI components are used for common UI elements.
*   Custom CSS is located in `resources/css/style.pcss` and processed by PostCSS (managed via `bun run dev` or `bun run build:css`).

### JavaScript

*   Minimal custom JavaScript is expected. HTMX handles most dynamic interactions.
*   JS assets are managed with Bun and built via `bun run build:js`. Entry point is `resources/js/app.ts`.

## Testing

*   **Go Tests:** Standard Go testing practices can be used for testing individual packages and functions.
*   **Playwright Tests:** End-to-end tests are written using Playwright and TypeScript.
    *   Test files are in `playwright-tests/`.
    *   Run tests using `bunx playwright test` (ensure the application is running).
    *   Update and add new E2E tests for significant UI changes or new features.

## General Guidelines

*   **Dependencies:**
    *   Go dependencies are managed with Go Modules (`go.mod`, `go.sum`). Run `go mod tidy` after adding/removing dependencies.
    *   Frontend dependencies are managed with Bun (`bun.lockb`, `package.json`).
*   **Code Style:**
    *   Follow standard Go formatting (`gofmt` or `goimports`).
    *   For frontend code, Prettier is configured (see `.prettierrc`).
*   **API Changes:** If you make changes to API endpoints or data structures that affect the frontend, ensure corresponding frontend code (Templ components, HTMX usage) is updated.
*   **Database Schema Changes:** When modifying PocketBase collections (tables), ensure migrations are correctly handled.
*   **Documentation:**
    *   Update this `AGENTS.MD` if there are significant changes to the development process, architecture, or key technologies.
    *   Comment Go code where necessary, especially for complex logic.

## Contact / Help

*   If you encounter issues or have questions about the development process, refer to the project's `README.md` and `CONTRIBUTING.md`.

---

*This document is intended for AI agents. Please update it as the project evolves.*
```
