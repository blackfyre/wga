# Web Gallery of Art

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fblackfyre%2Fwga.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fblackfyre%2Fwga?ref=badge_shield)
![CodeRabbit Pull Request Reviews](https://img.shields.io/coderabbit/prs/github/blackfyre/wga?utm_source=oss&utm_medium=github&utm_campaign=blackfyre%2Fwga&labelColor=171717&color=FF570A&link=https%3A%2F%2Fcoderabbit.ai&label=CodeRabbit+Reviews)

## Introduction

This repository contains the code for the Web Gallery of Art project. The project is a web application that allows users to browse through a collection of paintings, sculptures and other forms of Art. This project is intended to shave off the 3 decades of tech debt on the original website and provide a modern, responsive and user friendly experience with the same content.

## Activity

![Alt](https://repobeats.axiom.co/api/embed/9fd42cf5a4d13bf67b6ad9e58fe817130ebbf64f.svg "Repobeats analytics image")

## Technologies

The project is built around the following active technologies and workflows:

- [Go](https://go.dev/) 1.26.5 with [PocketBase](https://pocketbase.io) for the application server, data layer, hooks, and cron jobs
- [Templ](https://templ.guide/) for server-rendered UI fragments and page composition
- [Bun](https://bun.sh/) scripts for frontend dependency management and asset builds
- [PostCSS](https://postcss.org/) plus Tailwind tooling for stylesheet compilation
- [htmx](https://htmx.org) for incremental browser interactions
- [Playwright](https://playwright.dev/) for browser end-to-end coverage

## Getting Started

### Prerequisites

Copy `.env.example` to `.env` in the directory from which you start the application. `mise run app:init-env` creates it in the repository root for `mise run code:run`; copy it to `dist/.env` when using `mise run app:run`.

```bash
WGA_ENV=development

WGA_ADMIN_EMAIL=
WGA_ADMIN_PASSWORD=

WGA_S3_ENDPOINT=http://127.0.0.1:3900
WGA_S3_BUCKET=wga-assets
WGA_S3_REGION=garage
WGA_S3_ACCESS_KEY=GKlocaluploads
WGA_S3_ACCESS_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

WGA_PROTOCOL=http
WGA_HOSTNAME=localhost:8090

WGA_SMTP_HOST=127.0.0.1
WGA_SMTP_PORT=1025
WGA_SMTP_USERNAME=
WGA_SMTP_PASSWORD=
WGA_SENDER_ADDRESS=do-not-reply@wga.hu
WGA_SENDER_NAME=WGA
WGA_POSTCARD_FREQUENCY="*/1 * * * *"
WGA_RECAPTCHA_SITE_KEY=
WGA_RECAPTCHA_SECRET=

MAILPIT_URL=http://127.0.0.1:8025
```

| Variable                 | Description                                                                                      |
| ------------------------ | ------------------------------------------------------------------------------------------------ |
| `WGA_ENV`                | The environment the application is running in: `development`, `test`, `staging`, or `production` |
| `WGA_ADMIN_EMAIL`        | Optional email address for the bootstrap administrator                                           |
| `WGA_ADMIN_PASSWORD`     | Optional unique password for the bootstrap administrator                                         |
| `WGA_S3_ENDPOINT`        | The absolute S3-compatible object storage service endpoint                                       |
| `WGA_S3_BUCKET`          | The name of the S3 bucket                                                                        |
| `WGA_S3_REGION`          | The region of the S3 bucket                                                                      |
| `WGA_S3_ACCESS_KEY`      | The access-key ID for the S3-compatible object storage service                                   |
| `WGA_S3_ACCESS_SECRET`   | The access secret for the S3-compatible object storage service                                   |
| `WGA_PROTOCOL`           | The protocol to use for the application, valid values are `http` and `https`                     |
| `WGA_HOSTNAME`           | The domain pointing to the application                                                           |
| `WGA_SMTP_HOST`          | The address of the SMTP host                                                                     |
| `WGA_SMTP_PORT`          | The SMTP service port on the host address                                                        |
| `WGA_SMTP_USERNAME`      | The username for the SMTP service                                                                |
| `WGA_SMTP_PASSWORD`      | The password for the SMTP service                                                                |
| `WGA_SENDER_ADDRESS`     | The sending email address                                                                        |
| `WGA_SENDER_NAME`        | The name of the email sender                                                                     |
| `WGA_POSTCARD_FREQUENCY` | The five-field cron expression for sending queued postcards                                      |
| `WGA_RECAPTCHA_SITE_KEY` | The reCAPTCHA site key rendered in the postcard widget; required in staging and production       |
| `WGA_RECAPTCHA_SECRET`   | The reCAPTCHA secret used to verify postcard submissions; required in staging and production     |
| `MAILPIT_URL`            | The local Mailpit HTTP endpoint that Playwright queries during end-to-end tests                  |

Local `development` and `test` environments may omit `WGA_RECAPTCHA_SITE_KEY` and `WGA_RECAPTCHA_SECRET`; staging and production cannot start without both.

The administrator bootstrap is optional. Before the first application start, set both `WGA_ADMIN_EMAIL` and `WGA_ADMIN_PASSWORD` to unique values; leave both empty to skip it.

### Running the application

To run the application simply download the release for your platform and run it with:

```bash
./dist/wga serve
```

The application will start on port 8090 by default. You can access it by going to <http://localhost:8090>

### Build from source

The canonical build path uses [Mise](https://mise.jdx.dev/), which installs the pinned tools and defines the project tasks:

```bash
mise install
mise run app:build
```

This produces the server binary at `dist/wga`.

The equivalent build steps are:

```bash
mkdir -p dist
bun install
bun run build
templ generate
go mod tidy
go build -o dist/wga ./cmd/wga
```

Template sources live in `internal/assets/templ/`, generated `*_templ.go` files live beside those sources, and built frontend assets land in `internal/assets/public/`.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

### Development quick start

Start the local asset watchers and services with Mise:

```bash
mise run dev
```

`mise run dev` brings up the Podman Compose Mailpit and Garage services, then starts the frontend and template watchers. Mailpit exposes SMTP on port 1025 and its HTTP API on port 8025; Playwright reads `MAILPIT_URL` to query captured messages. Garage exposes S3-compatible storage on port 3900. In another terminal, start the application with `mise run code:run`, or run `mise run app:build` followed by `mise run app:run`. `mise run app:reset` brings up and waits for Garage while rebuilding and replacing `dist/wga_data`.

If you only need asset watchers, use the package scripts directly:

```bash
bun run build:watch:css
bun run build:watch:js
```

#### Synthetic bootstrap

The first application start on a fresh data directory applies the embedded synthetic-data migration. It imports records into the existing collections and attaches the bundled artwork and music files to the configured filesystem or S3-compatible storage.

The bootstrap migration skips an existing non-system application database rather than merging or replacing it. Changing the embedded source later requires a new migration; it does not rerun on an existing data directory. The development-only `seed:images` command remains available for placeholder-image generation.

## With Mise

Mise manages the project's development tools and tasks. Install Mise following its [getting-started guide](https://mise.jdx.dev/getting-started.html), then run:

```bash
mise install
mise run app:init-env
```

`mise.toml` defines the pinned tools, local environment defaults, build tasks, watchers, and Podman Compose local services.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fblackfyre%2Fwga.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fblackfyre%2Fwga?ref=badge_large)
