# Contributing

Thanks for helping improve `abp-cli`.

## Local Checks

```bash
go test ./...
go vet ./...
make build
make release-check
```

Run a dry preview when generator behavior changes:

```bash
abp-cli generate --root /path/to/abp-app --entity Product --dry-run
```

## Pull Requests

PR titles must follow Conventional Commits:

```text
feat(cli): add a new command
fix(generator): repair DbContext detection
docs(readme): clarify install steps
```

The PR workflow checks title format, `gofmt`, `go mod tidy`, `go vet`, tests, build, and installer script syntax.

## Issues

Use the bug or feature issue templates when possible. The issue workflow applies triage labels automatically based on the issue content.

## Releases

Release rules are documented in [.github/release-rules.md](.github/release-rules.md). Main branch pushes can create releases automatically when commits follow the release rules.

Every published version must pass `make release-check`. The release workflow runs that gate before it creates tags, builds archives, or publishes GitHub Releases.

Release changelog entries are generated from Conventional Commits. Use clear commit or squash-merge titles so `CHANGELOG.md` is useful to clients.
