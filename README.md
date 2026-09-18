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
* Supports `--help` / `-h` and `--version` / `-v`.
* Supports graceful cancellation while scanning.

## Installation

Download the latest binary from the [GitHub Releases](../../releases) page.

Jalebi currently provides Linux builds.

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
