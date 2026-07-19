<h1 align="center">ghosttag</h1>

<p align="center">
  <strong>Image metadata, made visible before you share.</strong>
</p>

<p align="center">
  A read-only, offline CLI that explains privacy-relevant metadata in JPEG and PNG files.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#what-you-get">Output</a> ·
  <a href="#notification-policy">Policy</a> ·
  <a href="SECURITY.md">Security</a>
</p>

---

`ghosttag` turns embedded image metadata into a compact, plain-English report. It identifies the file from its bytes, normalizes supported fields, and shows the context a recipient or platform may receive with the image.

> [!NOTE]
> `ghosttag` reports facts, not a risk score. Finding no supported metadata does not prove that an image is anonymous or safe to share.

## Quick Start

Clone the repository and build a Linux binary with the project-local Go toolchain:

```bash
git clone https://github.com/delriscotechnologies/ghosttag.git
cd ghosttag

bash ./scripts/bootstrap-go.sh
bash ./scripts/go-local.sh test ./...

mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  bash ./scripts/go-local.sh build -buildvcs=false -trimpath \
  -o ./bin/ghosttag ./cmd/ghosttag
```

Inspect one image:

```bash
./bin/ghosttag /path/to/image.jpg
```

The bootstrap script downloads the pinned official Go toolchain into this repository and verifies its published SHA-256 checksum. It does not install Go globally.

## What You Get

Given a synthetic PNG containing three repository-controlled metadata categories, `ghosttag` produces:

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

The report always includes the detected format, size, dimensions, and SHA-256 digest. Metadata values include their source container so the output remains traceable.

## How It Works

1. Read exactly one file without modifying it.
2. Detect JPEG or PNG from the file signature, not only its extension.
3. Calculate the SHA-256 digest and read image dimensions.
4. Parse supported EXIF, XMP, comment, and PNG text containers.
5. Normalize duplicate or equivalent tags into plain-language facts.
6. Group privacy-relevant facts into transparent categories.
7. Print a terminal-safe text report.

## Supported Metadata

| Format | Containers and fields |
| --- | --- |
| JPEG | APP1 EXIF, standard APP1 XMP, and COM comments |
| PNG | eXIf, tEXt, zTXt, iTXt, and XMP in `XML:com.adobe.xmp` |

Normalized facts can include location, capture time, device, software, authorship, copyright, comments, descriptions, and orientation.

## Notification Policy

The privacy-context section counts normalized categories, not raw tags:

| Categories found | Output behavior |
| --- | --- |
| 0 | State that no supported categories were found and clarify that this does not prove anonymity. |
| 1–2 | List the categories and facts without an elevated notice. |
| 3–5 | Add a gentle note that the combined details may reveal more context. |

The five counted categories are location, capture time, device, authorship, and comments. Software and orientation remain visible but do not affect the count. Several tags describing the same category count once.

The `3+` threshold controls product wording only. It is not a NIST metric, probability, severity level, or prediction of harm. The design follows NIST privacy guidance by separating observed data from contextual interpretation; see the [full notification policy](docs/notification-policy.md) for the definitions, rationale, and boundaries.

## Scope and Safeguards

| Control | Behavior |
| --- | --- |
| Read-only | Never removes, rewrites, or otherwise changes the image. |
| Offline inspection | Makes no network calls while analyzing a file. |
| Narrow scope | Accepts one JPEG or PNG file per execution; no recursive scans. |
| Byte-based detection | Warns when the extension disagrees with the detected content. |
| Terminal safety | Neutralizes control characters before printing embedded text. |
| Bounded parsing | Limits decompression of compressed PNG text. |
| No visual inference | Does not inspect pixels, subjects, faces, or scene content. |

Metadata may be missing, malformed, stale, or intentionally misleading. Standard JPEG XMP is supported, but extended multi-segment XMP is not reconstructed in this version. See [SECURITY.md](SECURITY.md) for the trust boundary and reporting guidance.

## Development in VS Code

The repository includes VS Code tasks that run through the default WSL distribution. Open the `ghosttag` folder, run **Tasks: Run Task**, and choose:

- `ghosttag: bootstrap Go`
- `ghosttag: test`
- `ghosttag: vet`
- `ghosttag: build Linux`

All downloaded tools, caches, temporary files, and binaries stay inside the repository and are ignored by Git. The implementation uses only the Go standard library. Additional design notes live in [`docs/`](docs/).

## License

Released under the [MIT License](LICENSE).
