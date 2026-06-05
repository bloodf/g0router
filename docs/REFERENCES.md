# Source References (Historical)

This document was used during the initial porting phase to map upstream source files to g0router target packages. The migration is complete. It is retained as an audit trail only.

The g0router codebase is now self-contained. There is no dependency on or requirement to consult external repositories. All relevant logic has been adapted, simplified, and integrated directly into the packages under `internal/` and `api/`.

For the current package layout see [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md).
For provider auth and capability details see [PROVIDERS.md](PROVIDERS.md).
For API contracts and SQLite schema see [SCHEMA.md](SCHEMA.md).
