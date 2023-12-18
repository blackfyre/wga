# Web Gallery of Art

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fblackfyre%2Fwga.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fblackfyre%2Fwga?ref=badge_shield)

## Introduction

This repositry contains the code for the Web Gallery of Art project. The project is a web application that allows users to browse through a collection of paintings, sculptures and other forms of Art. This project is inteded to shave off the 3 decades of tech debt on the original website and provide a modern, responsive and user friendly experience with the same content.

## Technologies

The project is built using the following technologies:

- [PocketBase](https://pocketbase.io) - A Go based SaaS platform for building web applications
- [htmx](https://htmx.org) - A javascript library for building web applications
- [Bulma](https://bulma.io) - A CSS framework for building responsive web applications
- [Go](https://go.dev/) 1.21+ - A programming language for building web applications
- [Sass](https://sass-lang.com/) - A CSS preprocessor for building responsive web applications
- [Goreleaser](https://goreleaser.com/) - A tool for building and releasing Go applications

## Getting Started

### Prerequisites

To run the application you'll have to have a `.env` file next to your executable with the following contents:

```bash {"id":"01HG08MCJXSNDZ9CYZ40JF0V9R"}
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
| `WGA_SMTP_PORT`        | The SMTP service port on the host adress                                                               |
| `WGA_SMTP_USERNAME`    | The username for the SMTP service                                                                      |
| `WGA_SMTP_PASSWORD`    | The password for the SMTP service                                                                      |
| `WGA_SENDER_ADDRESS`   | The sending email address                                                                              |
| `WGA_SENDER_NAME`      | The name of the email sender                                                                           |

### Running the application

To run the application simply download the release for your platform and run it with:

```bash {"id":"01HG08MCJYF04DTTTHQB8QKM5Z"}
./wga serve
```

The application will start on port 8090 by default. You can access it by going to <http://localhost:8090>

### Build from source

#### Prerequisites

To build the application you will need to have the following installed:

- [Go](https://go.dev/) 1.21+
- [NodeJS](https://nodejs.org/en/) 14+
- [NPM](https://www.npmjs.com/) 6+
- [Goreleaser](https://goreleaser.com/)

#### Building the application

Building the application relies on [Goreleaser](https://goreleaser.com/) to build the application. To build the application simply run:

```bash {"id":"01HG08MCJYF04DTTTHQC3TYW4M"}
goreleaser release --snapshot --clean
```

This will build the application and place the binary in the `dist` folder.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fblackfyre%2Fwga.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fblackfyre%2Fwga?ref=badge_large)
