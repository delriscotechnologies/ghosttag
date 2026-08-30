<h1 align="center">GHOSTTAG</h1>

<p align="center">
  A small read-only CLI for inspecting privacy-relevant JPEG and PNG metadata.
</p>

---

GHOSTTAG inspects one local image, calculates its SHA-256 hash, reports its dimensions, and extracts a focused set of metadata without modifying or uploading the file.

## Install

You need Linux and Go.

```bash
go install github.com/delriscotechnologies/ghosttag/cmd/ghosttag@latest
```

## What it does

- Detects JPEG or PNG from the image data and reports its dimensions.
- Calculates SHA-256 over the input bytes.
- Reads supported EXIF, XMP, JPEG comment, and PNG text metadata.
- Reports supported location, capture-time, device, authorship/copyright, and comment/description values.
- Groups those findings into privacy-relevant categories.
- Adds a context note when three or more categories are present.

## Output

Example output from a PNG containing XMP and text metadata:

```text
ghosttag — image metadata report

File
  Name: sample.png
  Detected format: PNG
  Size: 558 bytes
  Dimensions: 1 × 1 pixels
  SHA-256: 234e2aaecf8145e20ee32d6066321474877b031031878bb16ea5261883235bf2

Metadata
  Containers: PNG tEXt, PNG iTXt, PNG XMP
  Capture time:
    - 2026-07-18T12:34:56Z
  Device make:
    - Acme
  Device model:
    - Cam
  Author or copyright:
    - Alice
  Comment or description:
    - Hello world

Privacy context
  Categories found (4): capture time, device, authorship, comments
  Note: This file contains 4 privacy-relevant metadata categories: capture time, device, authorship, comments. In combination, these details can reveal more context than each detail alone. Consider whether they are appropriate for the intended recipient or platform.
```

## Demo

```bash
ghosttag /path/to/image.jpg
```

## Scope and limits

- Supported target: Linux.
- Accepts one regular JPEG or PNG file per run.
- Rejects files reported as larger than 100 MiB before reading.
- Makes no network calls during inspection and does not write to the input file.
- Does not decode pixels, recognize visual subjects, remove metadata, or scan directories.
- This is a focused metadata inspector, not a complete JPEG, PNG, TIFF, EXIF, or XMP implementation.
- JPEG metadata is read only before the first image scan.
- PNG `tEXt` values are decoded as Latin-1 and text keywords are matched case-sensitively.
- PNG `zTXt`, compressed `iTXt`, and XMP GPS are outside the current scope.
- EXIF GPS locations require rational DMS values with valid `N/S` and `E/W` direction references.
- Metadata may be missing, stale, malformed, or intentionally misleading.

See [SECURITY.md](SECURITY.md) for the security boundary.

## License

GHOSTTAG is available under the [MIT License](LICENSE).
