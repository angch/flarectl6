# TODO

## Setup & Analysis
- [x] Ensure `ref/` is populated (run `make fetch-ref`).
- [x] Analyze `ref/flarectl` structure (main entry point, command registration).
- [x] Create a strategy for handling configuration (API keys, output formats).

## Implementation
- [x] Implement `version` command.
- [x] Implement `zone` commands (list, create, details, delete).
- [x] Implement `dns` record commands.
- [ ] Implement `user` commands.
- [ ] ... (add more as we discover them in `ref/flarectl`).

## Documentation
- [ ] Keep `doc/flarectl-doc.md` updated with analysis.
- [ ] Keep `doc/cloudflare-go-doc.md` updated with library usage notes.
