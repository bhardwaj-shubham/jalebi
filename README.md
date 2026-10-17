# Jalebi

Jalebi is a small Go CLI for getting a quick overview of a software project.

It scans a project directory to report its structure, top programming languages, and developer notes such as `TODO`, `FIXME`, `BUG`, `REFACTOR`, `HACK`, and `NOTE`.

## Why we built it

Jalebi started as a small project for learning Go by building something useful rather than following isolated examples.

The project is also a way to explore software and systems concepts incrementally, including filesystem traversal, file I/O, streaming, cancellation, and concurrency as the project evolves.

## What it offers

* Validates the project path before scanning.
* Recursively counts files and directories.
* Skips common directories such as `.git`, `node_modules`, and `bin`.
* Detects developer-note markers in supported source and project files.
* Reports the top three programming languages found in the project.
* Shows the file, line, column, tag, and surrounding line for each detected note.
* Supports selecting the analysis type with `--type`.
* Supports ignoring files or directories with `--ignore`.
* Supports limiting tag occurrences with `--limit` (`0` means unlimited).
* Supports filtering developer-note markers with `--tag`.
* Supports filtering tag scans by language with `--lang`.
* Supports `--help` / `-h` and `--version` / `-v`.
* Supports graceful cancellation while scanning.

## Usage
Analyze a project with all available features:

```bash
jalebi /path/to/project
```

Run only a specific analysis:

```bash
jalebi --type files /path/to/project
jalebi --type languages /path/to/project
jalebi --type tags /path/to/project
```
The default analysis type is `all`, which runs all three analyses.

Ignore files or directories by name:

```bash
jalebi --ignore 'prisma|generated|dist' /path/to/project
```

Filter tags by type:

```bash
jalebi --type tags --tag 'TODO|FIXME' /path/to/project
```

Filter tag scanning by language:

```bash
jalebi --type tags --lang 'go|python' /path/to/project
```

Limit the number of matching tag occurrences:

```bash
jalebi --type tags --tag TODO --limit 10 /path/to/project
```

Combine filters:

```bash
jalebi --type tags --tag 'TODO|FIXME' --lang 'go|typescript' --limit 20 /path/to/project
```

The `--tag` and `--lang` options accept pipe-separated values. Values are case-insensitive, duplicate selections are ignored, and unsupported values are rejected. The `--limit` option defaults to `0`, meaning unlimited.

Language filtering applies only to tag scanning. With the default `all` analysis, file and language statistics remain complete even when a tag limit is reached.

Multiple ignore patterns can be provided as a single string separated by `|` (pipe) characters.

## Installation

Download the latest binary from the [GitHub Releases](../../releases) page.

Jalebi provides release binaries for multiple platforms, including Linux, macOS, and Windows.

After downloading, make the binary executable and move it to a directory in your `PATH` and run:

```bash
jalebi /path/to/project
```

## Build locally

### Requirements

* Go 1.27+

Clone the repository and run:

```bash
go run ./cmd/jalebi /path/to/project
```

Build a local binary:

```bash
go build -o jalebi ./cmd/jalebi
```

Run tests:

```bash
go test ./...
```

## Engineering decisions

Jalebi is intentionally kept small and incremental. Some of the implementation decisions include:

* **Single project traversal** - the available analyses are performed during a single `filepath.WalkDir` traversal rather than multiple traversals for each analysis type.
* **Streaming file reads** — use `bufio.Scanner` to process files line by line instead of loading complete files into memory.
* **Byte-based marker matching** — use byte searching and explicit word-boundary checks for fixed developer-note markers instead of introducing regular expressions.
* **Context-based cancellation** — pass `context.Context` through the analysis operations so a scan can stop cooperatively.
* **Focused file scanning** — only inspect known source/project file types when searching for developer notes rather than attempting to parse every file.
* **Automated releases** — GitHub Actions and GoReleaser handle reproducible release builds while keeping the release implementation in repository configuration.

## Issues & contributions

Found a bug or have an idea for improving Jalebi?

Open an issue or submit a pull request on GitHub.

## License

Jalebi is licensed under the MIT [LICENSE](LICENSE).
