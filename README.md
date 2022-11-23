# The Web Gallery of Art

This project is a an attempt to preserve the knowledge gathered by Emil Krén at https://www.wga.hu/ in a more modern format.

## Why?

The original site is a great resource, but it is not easy to use. The site is not responsive, and the images are not optimized for mobile devices. The site is also not easy to navigate, and the search function is not very useful.

## How?

The site ib built with [SvelteKit](https://kit.svelte.dev/) and [IBM Carbon Design System](https://carbondesignsystem.com/). The backend is built on [PocketBase](https://pocketbase.io/).

## How to contribute?

If you want to contribute, you can do so by forking the repository and making a pull request.

## How to run locally?

### Prerequisites

- [Go 1.19+](https://golang.org/dl/)(optional, pre-built binary already included)
- [Node Version Manager](https://github.com/nvm-sh/nvm)
  - [Node.js v18](https://nodejs.org/en/download/)
  - [PNPM](https://pnpm.io/installation)

### Steps

```bash
# Clone the repository
git clone git@github.com:blackfyre/wga.git

# Change directory
cd wga

# Set node version
nvm use

# Install dependencies
pnpm install

# Build Frontend
pnpm build

# Run Backend
pnpm serve

```

The site should now be running on http://localhost:8090
