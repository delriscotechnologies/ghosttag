# ghosttag design

## Product contract

`ghosttag` accepts one path to a JPEG or PNG file, performs an offline read-only inspection, writes one English terminal report, and exits.

Exit codes:

| Code | Meaning |
| --- | --- |
| 0 | The report or requested help/version output was written successfully. |
| 1 | The file could not be read, parsed, or reported. |
| 2 | The command-line input was invalid. |

## Data flow

```text
one file path
    |
    v
file facts + SHA-256
    |
    v
byte-signature format detection
    |
    +--> JPEG markers --> EXIF / XMP / COM
    |
    +--> PNG chunks ----> eXIf / tEXt / zTXt / iTXt / XMP
    |
    v
normalized metadata model
    |
    +--> factual terminal sections
    |
    +--> privacy-relevant category grouping
             |
             v
       optional 3+ context note
```

Parsing and reporting are separated so future output formats can consume the same normalized model without changing the notification policy.

## Package responsibilities

- `internal/inspect` owns file reading, format detection, container parsing, normalization, deduplication, and parser warnings.
- `internal/model` contains data structures without parsing or presentation logic.
- `internal/assessment` maps normalized facts to categories. It has no file-format knowledge.
- `internal/report` renders the model as deterministic terminal text.
- `internal/cli` owns arguments, help, errors, and process exit codes.
- `cmd/ghosttag` is a minimal executable entry point.

## Parser behavior

JPEG inspection stops before entropy-coded image data. It reads marker lengths, APP1 EXIF/XMP payloads, COM comments, and start-of-frame dimensions.

PNG inspection walks bounded chunks, checks each CRC, reads dimensions from IHDR, and processes supported metadata chunks. zTXt and compressed iTXt values use zlib with a one-megabyte decompressed-text limit.

EXIF parsing supports little- and big-endian TIFF directories, primary IFD fields, EXIF subdirectories, and GPS subdirectories. Invalid metadata produces a warning when the surrounding image structure can still be inspected.

XMP parsing uses XML namespaces by local field name and supports both element values and common attribute values. Structured RDF creator, rights, and description lists are normalized from their list items.

## Output safety

Metadata is untrusted input. Before display, control characters are replaced with spaces, whitespace is collapsed, and each textual value is limited to 500 Unicode code points. This prevents embedded terminal control sequences from being replayed to the user.

The inspector never writes to the input file.

## Local development environment

All generated development state is routed into the repository:

| State | Location |
| --- | --- |
| Linux Go toolchain | `.tools/go` |
| Installed Go tools | `.tools/bin` |
| Build cache | `.cache/go-build` |
| Module cache | `.cache/go-mod` |
| Temporary files | `.tmp` |
| Built binary | `bin/ghosttag` |
| Coverage or release artifacts | `coverage` and `dist` |

The current implementation has no third-party Go modules.
