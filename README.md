# Web Gallery of Art

## Introduction

This repositry contains the code for the Web Gallery of Art project. The project is a web application that allows users to browse through a collection of paintings, sculptures and other forms of Art. This project is inteded to shave off the 3 decades of tech debt on the original website and provide a modern, responsive and user friendly experience with the same content.

## Technologies

The project is built using the following technologies:

- [PocketBase](https://pocketbase.io) - A Go based SaaS platform for building web applications
- [htmx](https://htmx.org) - A javascript library for building web applications
- [Bulma](https://bulma.io) - A CSS framework for building responsive web applications
- [Go](https://go.dev/) 1.18+ - A programming language for building web applications
- [Sass](https://sass-lang.com/) - A CSS preprocessor for building responsive web applications
- [Goreleaser](https://goreleaser.com/) - A tool for building and releasing Go applications

## Getting Started

### Running the application

To run the application simply download the release for your platform and run it with:

```bash
./wga serve
```

or if you are on windows:

```bash
wga.exe serve
```

The application will start on port 8090 by default. You can access it by going to <http://localhost:8090>

### Build from source

#### Prerequisites

To build the application you will need to have the following installed:

- [Go](https://go.dev/) 1.18+
- [NodeJS](https://nodejs.org/en/) 14+
- [NPM](https://www.npmjs.com/) 6+
- [Goreleaser](https://goreleaser.com/)

#### Building the application

Building the application relies on [Goreleaser](https://goreleaser.com/) to build the application. To build the application simply run:

```bash
goreleaser release --snapshot --clean
```

This will build the application and place the binary in the `dist` folder.

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for more information.

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
