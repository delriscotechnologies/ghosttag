<h1 align="center">GHOSTTAG</h1>

<p align="center">
  A read-only, offline tool that extracts metadata from JPEG and PNG files and generates a report.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> ·
  <a href="#the-report">Report</a> ·
  <a href="#metadata-coverage">Metadata</a> ·
  <a href="SECURITY.md">Security</a>
</p>

---

GHOSTTAG inspects one JPEG or PNG file and extracts supported EXIF, XMP, comment, and PNG text metadata into a terminal report.

Each run reports the file details, metadata values, source containers, privacy-relevant categories, and warnings when needed. It does not upload the image, modify the file, analyze visual content, or scan directories.

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

To build and install directly from the repository instead:

```bash
git clone https://github.com/delriscotechnologies/ghosttag.git
cd ghosttag
go build -trimpath -o ./bin/ghosttag ./cmd/ghosttag
sudo install -m 0755 ./bin/ghosttag /usr/local/bin/ghosttag
```

## The Report

Every report includes three core sections. A **Warnings** section is added only when malformed data, an extension mismatch, or a safety limit needs to be reported.

| Section | Contents |
| --- | --- |
| **File** | Detected format, extension, size, dimensions, and SHA-256 |
| **Metadata** | Extracted values and every supported source container that supplied each value |
| **Warnings** | Present only when warnings were generated |
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

The example is generated from repository-controlled synthetic report data and checked by an automated test.

</details>

## Metadata Coverage

| Format | Supported containers |
| --- | --- |
| **JPEG** | APP1 EXIF, standard namespace-aware APP1 XMP, and COM comments across supported scans |
| **PNG** | eXIf, tEXt, zTXt, iTXt, and namespace-aware XMP stored as `XML:com.adobe.xmp` |

Extracted values can include GPS location, validated capture time, device make and model, software, authorship, copyright information, comments, descriptions, captions, and orientation.

Standard JPEG XMP is supported. Extended multi-segment JPEG XMP is not reconstructed. Metadata can also be missing, malformed, stale, or intentionally misleading. Values using unknown XMP namespaces are ignored rather than treated as standard properties.

## How It Works

1. Opens one regular file, rejects symbolic-link inputs, and verifies opened-file identity where atomic no-follow support is unavailable.
2. Rejects directories, devices, FIFOs, other special files, files larger than 100 MiB, and files whose size or modification time changes during reading.
3. Detects JPEG or PNG from the file signature instead of trusting the extension.
4. Calculates the SHA-256 digest and reads validated image dimensions.
5. Bounds JPEG marker traversal, validates complete frame headers, and continues safely across multiple scans.
6. Requires valid CRCs for critical PNG chunks and ignores ancillary chunks whose CRC is invalid.
7. Parses supported metadata containers within explicit per-field and parser limits.
8. Groups equivalent fields while retaining every distinct source container.
9. Validates capture times and GPS directions and components before assigning privacy categories.
10. Neutralizes unsafe terminal characters in reports and diagnostics.

The parser validates only the container structure needed to locate supported metadata and dimensions. It is not a full image decoder and does not inspect pixels, recognize visual subjects, or inspect faces.

## Privacy Context

The report groups supported findings into five categories:

| Category | Included findings |
| --- | --- |
| **Location** | Supported GPS coordinates |
| **Capture time** | Validated original, digitized, or creation timestamps |
| **Device** | Camera or capture-device make and model |
| **Authorship** | Author, artist, owner, or copyright values |
| **Comments** | Comments, descriptions, captions, or free-form notes |

The category count changes report wording only. Zero categories does not prove anonymity, and three or more categories does not represent a severity score, probability, or prediction of harm. See the [notification policy](docs/notification-policy.md) for the complete rules.

## Scope and Safeguards

| Boundary | Enforcement |
| --- | --- |
| **File access** | Rejects symbolic-link inputs, verifies opened-file identity, detects common concurrent changes, and never writes to the file |
| **Input size** | Rejects files larger than 100 MiB before parsing |
| **Parsing** | Limits JPEG markers, PNG chunks, metadata size, decompression, XMP depth and tokens, warnings, locations, and values per metadata field |
| **Integrity** | Validates JPEG frame headers, enforces PNG critical CRCs, skips invalid ancillary chunks, and keeps XMP GPS pairs within their descriptions |
| **Coordinates and time** | Validates GPS direction and components and accepts only recognized capture-time formats |
| **Terminal output** | Replaces control and Unicode format characters and truncates long text values in reports and diagnostics |
| **Network** | Makes no network calls during inspection |
| **Scope** | Accepts one JPEG or PNG per execution and does not scan directories |

It does not remove metadata, determine whether metadata is true, or decide whether an image is safe to share. Concurrent writers can still attempt unusual races, so inspect an unchanged copy of hostile input with minimum privileges and operating-system resource limits. See [SECURITY.md](SECURITY.md) for the trust boundary and vulnerability-reporting process.

## License

Available under the [MIT License](LICENSE).
