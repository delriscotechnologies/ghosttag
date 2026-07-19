# GHOSTTAG

Extract image metadata and generate a local report.

**Read-only · Offline · JPEG and PNG · Linux**

[Quick Start](#quick-start) · [Report](#the-report) · [Metadata](#metadata-coverage) · [Security](SECURITY.md)

Give it one JPEG or PNG file. It extracts supported metadata and reports the file details, metadata values, source containers, warnings, and privacy-relevant categories it finds.

| | |
| --- | --- |
| **Input** | One JPEG or PNG file |
| **Output** | A terminal report |
| **Network access** | None during inspection |
| **File changes** | None |

## Quick Start

Go 1.26 or newer is required.

Install the command:

```bash
go install github.com/delriscotechnologies/ghosttag/cmd/ghosttag@latest
```

Make sure `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset, is included in your `PATH`. Then inspect an image:

```bash
ghosttag /path/to/image.jpg
```

## The Report

Every report is divided into four parts:

| Section | Contents |
| --- | --- |
| **File** | Detected format, extension, size, dimensions, and SHA-256 |
| **Metadata** | Extracted values and the container each value came from |
| **Warnings** | Malformed data, extension mismatches, and safety-limit omissions |
| **Privacy context** | Supported privacy categories found in the file |

<details>
<summary><strong>View an example report</strong></summary>

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

The example uses synthetic metadata stored in a repository-controlled test image.

</details>

## Metadata Coverage

| Format | Supported containers |
| --- | --- |
| **JPEG** | APP1 EXIF, standard APP1 XMP, and COM comments |
| **PNG** | eXIf, tEXt, zTXt, iTXt, and XMP stored as `XML:com.adobe.xmp` |

Extracted values can include GPS location, capture time, device make and model, software, authorship, copyright information, comments, descriptions, captions, and orientation.

Standard JPEG XMP is supported. Extended multi-segment JPEG XMP is not reconstructed. Metadata can also be missing, malformed, stale, or intentionally misleading.

## How It Works

1. Opens one regular file without following symbolic links.
2. Rejects directories, devices, FIFOs, other special files, and files larger than 100 MiB.
3. Detects JPEG or PNG from the file signature instead of trusting the extension.
4. Calculates the SHA-256 digest and reads the image dimensions.
5. Parses supported metadata containers within explicit limits.
6. Groups equivalent fields while retaining their source container.
7. Neutralizes unsafe terminal characters and prints the report.

The parser reads metadata containers only. It does not decode pixels, recognize visual subjects, or inspect faces.

## Privacy Context

The report groups supported findings into five categories:

| Category | Included findings |
| --- | --- |
| **Location** | Supported GPS coordinates |
| **Capture time** | Original, digitized, or creation timestamps |
| **Device** | Camera or capture-device make and model |
| **Authorship** | Author, artist, owner, or copyright values |
| **Comments** | Comments, descriptions, captions, or free-form notes |

The category count changes report wording only. Zero categories does not prove anonymity, and three or more categories does not represent a severity score, probability, or prediction of harm. See the [notification policy](docs/notification-policy.md) for the complete rules.

## Scope and Safeguards

| Boundary | Enforcement |
| --- | --- |
| **File access** | Opens one regular file and never writes to it |
| **Input size** | Rejects files larger than 100 MiB before parsing |
| **Parsing** | Limits chunks, metadata size, decompression, XMP depth and tokens, warnings, and normalized values |
| **Coordinates** | Rejects `NaN`, infinity, and out-of-range GPS values |
| **Terminal output** | Replaces control and Unicode format characters and truncates long text values |
| **Network** | Makes no network calls during inspection |
| **Scope** | Accepts one JPEG or PNG per execution and does not scan directories |

It does not remove metadata, determine whether metadata is true, or decide whether an image is safe to share. Inspect hostile files with minimum privileges and operating-system resource limits. See [SECURITY.md](SECURITY.md) for the trust boundary and vulnerability-reporting process.

## Install from Source

The repository uses the Go installation already available on the system. It does not download or maintain a separate toolchain.

```bash
git clone https://github.com/delriscotechnologies/ghosttag.git
cd ghosttag

go test ./...
go build -trimpath -o ./bin/ghosttag ./cmd/ghosttag
sudo install -m 0755 ./bin/ghosttag /usr/local/bin/ghosttag
```

## Development

```bash
go fmt ./...
go test ./...
go vet ./...
go build -trimpath -o ./bin/ghosttag ./cmd/ghosttag
```

The same workflow is available through `make check`, `make build`, and `sudo make install`. CI tests the code, runs `go vet`, verifies standard Go installation, and builds Linux AMD64 and ARM64 binaries. The implementation uses only the Go standard library.

## License

Available under the [MIT License](LICENSE).
