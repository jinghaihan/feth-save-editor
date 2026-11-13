# Building and releasing

`VERSION` is the application release number. The supported game version is 1.2.0.

The application and CLI use Go 1.27.3 or later. Run `go test -race ./...` and `go vet ./...` to verify the save engine and standard MyGo controls. All fixtures are synthetic; tests never read user saves.

Build a platform with `python3 scripts/build_go.py osx-arm64`, replacing the target with `win-x64`, `osx-x64`, or `linux-x64` as needed. Package it with `python3 scripts/package_release.py osx-arm64`, then run `python3 scripts/test_release_package.py osx-arm64` on the matching OS. macOS packaging uses the system `sips`, `iconutil`, `codesign`, and `ditto` tools. Packages include the GUI, CLI, original application icon, and dependency licenses.

Every push to `main` or `refactor/mygo`, pull request, or manual run of the [build workflow](../.github/workflows/build.yml) builds, tests, packages and re-tests extracted ZIPs on Windows, Apple Silicon macOS, Intel macOS, and Linux. `feth-mygo-*` artifacts are retained for 14 days.

To publish from a clean, synchronized `main` branch, run `python3 scripts/release.py` and choose patch, minor, or major. A direct type such as `python3 scripts/release.py patch` is also supported. The script updates `VERSION`, commits it, pushes `main`, and pushes an annotated tag. Use `--check` to preview without publishing. Validation and publication run in CI.

The [release workflow](../.github/workflows/release.yml) tests the tagged source on all four targets, packages the downloads, generates release notes with changelogithub, uploads platform ZIPs and SHA-256 hashes, and downloads them again to verify those hashes. Creating a branch build does not publish a release.
