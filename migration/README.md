# Migration regression oracle

`legacy/core` and `legacy/cli` are the frozen C# v0.3.2 implementation, used only as a regression oracle. They are not built or shipped with the Go application. `legacy/ui-localization` retains the existing UI translations used by the data exporter.

The development-only exporter regenerates the embedded packed layouts and catalogs from this oracle:

```sh
dotnet run --project tools/export-legacy/Export.csproj -c Release -- internal/catalog/data
```

Compare the old and new editors using generated saves only:

```sh
dotnet build migration/legacy/cli/Cli.csproj -c Release
go build -o build/FethEditor.Cli ./cmd/feth-cli
python3 scripts/test_migration.py --legacy migration/legacy/cli/bin/Release/net10.0/FethEditor.Cli.dll --go build/FethEditor.Cli
```

The comparison checks all 12 languages, catalogs, full inspect JSON, all edit operations, byte-level differences, output hashes, and version-5/version-7 system saves. Normal Go builds and tests do not require .NET or the oracle.
