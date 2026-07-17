# Web Gallery of Art

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fblackfyre%2Fwga.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fblackfyre%2Fwga?ref=badge_shield)

## Introduction

This repository contains the code for the Web Gallery of Art project. The project is a web application that allows users to browse through a collection of paintings, sculptures and other forms of Art. This project is intended to shave off the 3 decades of tech debt on the original website and provide a modern, responsive and user friendly experience with the same content.

## Activity

![Alt](https://repobeats.axiom.co/api/embed/9fd42cf5a4d13bf67b6ad9e58fe817130ebbf64f.svg "Repobeats analytics image")

## Technologies

The project is built around the following active technologies and workflows:

- [Go](https://go.dev/) 1.24+ with [PocketBase](https://pocketbase.io) for the application server, data layer, hooks, and cron jobs
- [Templ](https://templ.guide/) for server-rendered UI fragments and page composition
- [Bun](https://bun.sh/) scripts for frontend dependency management and asset builds
- [PostCSS](https://postcss.org/) plus Tailwind tooling for stylesheet compilation
- [htmx](https://htmx.org) for incremental browser interactions
- [Playwright](https://playwright.dev/) for browser end-to-end coverage

## Getting Started

### Prerequisites

To run the application you'll have to have a `.env` file next to your executable with the following contents:

```bash
WGA_ENV=development

WGA_ADMIN_EMAIL=
WGA_ADMIN_PASSWORD=

WGA_S3_ENDPOINT=
WGA_S3_BUCKET=
WGA_S3_REGION=
WGA_S3_ACCESS_KEY=
WGA_S3_ACCESS_SECRET=

WGA_PROTOCOL=http
WGA_HOSTNAME=localhost:8090

WGA_SMTP_HOST=
WGA_SMTP_PORT=
WGA_SMTP_USERNAME=
WGA_SMTP_PASSWORD=
WGA_SENDER_ADDRESS=
WGA_SENDER_NAME=
WGA_RECAPTCHA_SECRET=

MAILPIT_URL=
```

| Variable               | Description                                                                                            |
| ---------------------- | ------------------------------------------------------------------------------------------------------ |
| `WGA_ENV`              | The environment the application is running in, valid values are `development`, `staging`, `production` |
| `WGA_ADMIN_EMAIL`      | The email address of the admin user                                                                    |
| `WGA_ADMIN_PASSWORD`   | The password of the admin user                                                                         |
| `WGA_S3_ENDPOINT`      | The endpoint of the S3 bucket                                                                          |
| `WGA_S3_BUCKET`        | The name of the S3 bucket                                                                              |
| `WGA_S3_REGION`        | The region of the S3 bucket                                                                            |
| `WGA_S3_ACCESS_KEY`    | The access key of the S3 bucket                                                                        |
| `WGA_S3_ACCESS_SECRET` | The access secret of the S3 bucket                                                                     |
| `WGA_PROTOCOL`         | The protocol to use for the application, valid values are `http` and `https`                           |
| `WGA_HOSTNAME`         | The domain pointing to the application                                                                 |
| `WGA_SMTP_HOST`        | The address of the SMTP host                                                                           |
| `WGA_SMTP_PORT`        | The SMTP service port on the host address                                                              |
| `WGA_SMTP_USERNAME`    | The username for the SMTP service                                                                      |
| `WGA_SMTP_PASSWORD`    | The password for the SMTP service                                                                      |
| `WGA_SENDER_ADDRESS`   | The sending email address                                                                              |
| `WGA_SENDER_NAME`      | The name of the email sender                                                                           |
| `WGA_RECAPTCHA_SECRET` | The reCAPTCHA secret used to verify postcard submissions                                               |
| `MAILPIT_URL`          | The local mail UI endpoint that Playwright checks during end-to-end tests                              |

### Running the application

To run the application simply download the release for your platform and run it with:

```bash
./dist/wga serve
```

The application will start on port 8090 by default. You can access it by going to <http://localhost:8090>

### Build from source

The canonical build path uses `devenv`:

```bash
devenv shell
app:build
```

This produces the server binary at `dist/wga`.

If you need the raw commands outside `devenv`, run:

```bash
mkdir -p dist
bun install
bun run build
templ generate
go build -o dist/wga ./cmd/wga
```

Template sources live in `internal/assets/templ/`, generated `*_templ.go` files live beside those sources, and built frontend assets land in `internal/assets/public/`.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

### Development quick start

Use `devenv` for the normal development loop:

```bash
devenv shell
devenv up
```

`devenv up` starts the frontend watchers, template watcher, the local `mailhog` service, and MinIO. Playwright reads `MAILPIT_URL` as the browser endpoint for inspecting those captured messages. To run the application server, either build and launch the binary with `app:run` or start it directly with `code:run`.

If you only need asset watchers, use the package scripts directly:

```bash
bun run build:watch:css
bun run build:watch:js
```

#### Seeding

The database is populated on first start, and if you want to have images available, make sure that your `WGA_ENV=development` is set and then you can execute:

```bash
./dist/wga seed:images
```

This will go through the contents of the database and will use placeholder images to "generate" the necessary images to the designated S3 compatible file hosting solution designated in the `.env` file.

## With DevEnv

Devenv is a Nix based development environment that's more easily accessible than a pure nix based approach.  
You can check [devenv.sh](https://devenv.sh/getting-started/) for installation and usage details.

For first-time setup:

1. Install Devenv following the [getting started guide](https://devenv.sh/getting-started/)
2. (optional) copy the `devenv.local.stub.nix` file to `devenv.local.nix`
3. Run `devenv up` to start the development environment

Checking the [/devenv.nix](devenv.nix) file for more details is also recommended.
This file contains:

- Development dependencies
- Environment variables
- Service configurations

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fblackfyre%2Fwga.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fblackfyre%2Fwga?ref=badge_large)
