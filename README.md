<h1 align="center">GHOSTTAG</h1>

<p align="center">
  A read-only Linux tool for inspecting JPEG and PNG metadata.
</p>

---

GHOSTTAG extracts supported EXIF, XMP, comment, and PNG text metadata from one image and produces a terminal report.

It does not upload or modify the image.

## Install

You need Linux and Go.

```bash
go install github.com/delriscotechnologies/ghosttag/cmd/ghosttag@latest
```

Then inspect an image:

```bash
ghosttag /path/to/image.jpg
```

## What it does

1. Detects JPEG or PNG from the file signature.
2. Calculates the SHA-256 hash and image dimensions.
3. Extracts supported metadata.
4. Groups privacy-relevant findings such as location, capture time, device, authorship, and comments.
5. Reports malformed or unsupported data as warnings when needed.

## Output

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
  Note: This file contains 3 privacy-relevant metadata categories.
```

## Demo

```bash
ghosttag photo.jpg
```

## Scope and limits

- Linux only.
- Accepts one JPEG or PNG file per run.
- Maximum input size is 100 MiB.
- Read-only and makes no network calls during inspection.
- Does not scan directories, inspect pixels, recognize faces, or remove metadata.
- Metadata can be missing, malformed, stale, or intentionally misleading.

See [SECURITY.md](SECURITY.md) for security guidance.

## License

GHOSTTAG is available under the [MIT License](LICENSE).
