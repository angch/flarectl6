This project is a reimplementation of flarectl using the github.com/cloudflare/cloudflare-go/v6 library.

The older flarectl is a command line version available at github.com/cloudflare/cloudflare-go/tree/v0.116.0/cmd/flarectl.
We want to replicate the command line functionality using the modern v6 library which supports newer features and likely has a different API structure.

## Setup

To analyze the differences and guide the reimplementation, we use a local reference of both the legacy code and the new library.

Run the following to fetch the reference sources into `ref/`:
```sh
make fetch-ref
```

This will populate:
- `ref/flarectl`: Source of the legacy command line tool.
- `ref/cloudflare-go`: Source of the v6 library.
