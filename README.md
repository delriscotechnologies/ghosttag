<h1 align="center">GHOSTTAG</h1>

<p align="center">
  <strong>One image. Hidden context. A readable report.</strong>
</p>

<p align="center">
  A read-only, offline CLI that explains privacy-relevant metadata in JPEG and PNG files.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#how-ghosttag-works">How It Works</a> ·
  <a href="#inside-the-report">Report</a> ·
  <a href="SECURITY.md">Security</a>
</p>

---

GHOSTTAG shows what an image can reveal beyond its pixels. Give it one JPEG or PNG and it identifies the real format, calculates a SHA-256 digest, reads supported metadata containers, and turns the findings into a compact terminal report.

The inspection stays local. GHOSTTAG does not upload the image, modify the file, analyze visual content, or crawl directories.

> [!NOTE]
> GHOSTTAG reports observable metadata, not a risk score. Finding no supported metadata does not prove that an image is anonymous or safe to share.

## Quick Start

GHOSTTAG supports native Linux on AMD64 and ARM64. The bootstrap requires Bash, `curl`, `tar`, and `sha256sum`.

```bash
git clone https://github.com/delriscotechnologies/ghosttag.git
cd ghosttag

bash ./scripts/bootstrap-go.sh
mkdir -p bin
CGO_ENABLED=0 GOOS=linux \
  bash ./scripts/go-local.sh build -buildvcs=false -trimpath \
  -o ./bin/ghosttag ./cmd/ghosttag
```

Inspect one image:

```bash
./bin/ghosttag /path/to/image.jpg
```

The bootstrap downloads the matching pinned Go toolchain from `go.dev`, verifies its published SHA-256 checksum, and recreates `.tools/go` from the verified archive. It does not install Go globally or trust an existing ignored toolchain directory.

## How GHOSTTAG Works

An inspection moves through eight bounded stages:

1. Open exactly one regular file without following symbolic links.
2. Reject directories, devices, FIFOs, other special files, and files larger than 100 MiB.
3. Detect JPEG or PNG from the file signature rather than trusting the extension.
4. Calculate the SHA-256 digest and read the image dimensions.
5. Parse supported EXIF, XMP, comment, and PNG text containers.
6. Normalize equivalent tags into plain-language facts and retain their source container.
7. Group privacy-relevant facts into transparent categories.
8. Neutralize unsafe terminal characters and print the report.

The parser never decodes image pixels. A renamed JPEG still reports as JPEG, and an extension mismatch appears as a warning.

## Inside the Report

A synthetic PNG containing capture time, author, and comment metadata produces:

```text
ghosttag — image metadata report

File
  Name: sample-metadata.png
  Detected format: PNG
  Extension: .png
  Size: 402 B (402 bytes)
  Dimensions: 2 × 2 pixels
  SHA-256: 2f19e2d43cf22ae8ffe544505f01e30e03cffbbd02075ee7e0afcf3bf78fed63

Metadata
  Containers: PNG tEXt, PNG iTXt, PNG XMP
  Capture time:
    - 2026-07-18T12:34:56Z [PNG XMP]
  Author:
    - Example Author [PNG tEXt:Author]
  Comment or description:
    - Example comment [PNG tEXt:Comment]

Privacy context
  Categories found (3): capture time, authorship, comments
  Note: This file contains 3 privacy-relevant metadata categories: capture time, authorship, comments. In combination, these details can reveal more context than each detail alone. Consider whether they are appropriate for the intended recipient or platform.
```

Each section answers a different question:

| Section | What it tells you |
| --- | --- |
| File | Actual format, extension, size, dimensions, and SHA-256 |
| Metadata | Normalized values and the container each value came from |
| Warnings | Malformed metadata, extension mismatches, or omitted data that reached a safety limit |
| Privacy context | Which supported categories were present and whether several appeared together |

## Metadata Coverage

| Format | Supported containers |
| --- | --- |
| JPEG | APP1 EXIF, standard APP1 XMP, and COM comments |
| PNG | eXIf, tEXt, zTXt, iTXt, and XMP stored as `XML:com.adobe.xmp` |

Normalized findings can include:

- GPS location
- capture time
- device make and model
- editing or capture software
- author and copyright information
- comments, descriptions, and captions
- orientation

Metadata can be missing, malformed, stale, or intentionally misleading. Standard JPEG XMP is supported, but extended multi-segment XMP is not reconstructed.

## Privacy Context

GHOSTTAG counts normalized categories, not raw tags. Five device-related tags still count as one `device` category.

| Category | Included findings |
| --- | --- |
| Location | Supported GPS coordinates |
| Capture time | Original, digitized, or creation timestamps |
| Device | Camera or capture-device make and model |
| Authorship | Author, artist, owner, or copyright values |
| Comments | Comments, descriptions, captions, or free-form notes |

The category count controls wording only:

| Categories found | Report behavior |
| --- | --- |
| 0 | States that no supported categories were found and clarifies that this does not prove anonymity |
| 1–2 | Lists the categories and facts without an elevated notice |
| 3–5 | Adds a gentle note that the combined details may reveal more context |

The `3+` threshold is a GHOSTTAG product policy. It is not a NIST metric, probability, severity level, or prediction of harm. See the [notification policy](docs/notification-policy.md) for definitions, rationale, and boundaries.

## Scope and Safeguards

GHOSTTAG is deliberately narrow:

| Boundary | Enforcement |
| --- | --- |
| File access | Opens one regular file, rejects symbolic links and Linux special files, and never writes to the input |
| Input size | Rejects files larger than 100 MiB before parsing |
| Parsing | Limits PNG chunks, metadata chunk size, decompression, XMP nesting and tokens, warnings, and normalized values |
| Coordinates | Rejects `NaN`, infinity, and out-of-range GPS values |
| Terminal output | Replaces control and Unicode format characters before display and truncates long text values |
| Network | Makes no network calls while inspecting an image |
| Scope | Accepts one JPEG or PNG per execution and does not scan directories recursively |

GHOSTTAG does not remove metadata, inspect pixels, recognize faces or objects, determine whether metadata is true, or decide whether sharing an image is safe.

Malformed files can still expose implementation defects. Inspect hostile files with minimum privileges and operating-system resource limits. See [SECURITY.md](SECURITY.md) for the trust boundary and vulnerability-reporting process.

## Development

Run the complete local workflow with the repository-managed Go toolchain:

```bash
make bootstrap
make check
```

Individual commands are also available:

```bash
make test
make vet
make build
```

CI runs the test and vet suites and builds Linux AMD64 and ARM64 binaries. The repository also includes native Linux VS Code tasks for bootstrap, test, vet, and build operations.

All downloaded tools, caches, temporary files, and binaries remain inside the repository and are ignored by Git. The implementation uses only the Go standard library. Additional design notes are available in [`docs/`](docs/).

## License

GHOSTTAG is available under the [MIT License](LICENSE).
