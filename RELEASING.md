# Releasing

This repository ships Windows releases through GitHub Actions.

## Release flow

1. Update [`CHANGELOG.md`](CHANGELOG.md).
2. Make sure the release commit contains only the changes you want to ship.
3. Create the version tag in `vX.Y.Z` form.
4. Push `main` and the tag.
5. Let [`.github/workflows/release.yml`](.github/workflows/release.yml) run.
6. Verify the GitHub release page contains:
   - `wtm-X.Y.Z-windows-amd64.exe`
   - `wtm-X.Y.Z-windows-amd64.exe.sha256`

## Local build

You can produce the same style of binary locally:

```powershell
.\build.ps1 -Version 0.1.0 -Out wtm-0.1.0-windows-amd64.exe
```

## Notes

- The workflow runs `go vet ./...`, `go test ./...`, then builds the Windows `.exe`.
- The version is injected with `-X main.version=X.Y.Z`.
- Versioned release binaries are ignored by Git through [`.gitignore`](.gitignore).
- Keep the working tree clean before tagging so the release matches the intended source state.
