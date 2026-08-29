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

The terminal report includes:

| Section | Contents |
| --- | --- |
| File | Format, size, dimensions, and SHA-256 |
| Metadata | Extracted metadata and source containers |
| Warnings | Parsing or validation warnings when present |
| Privacy context | Privacy-relevant categories found |

## Demo

```bash
ghosttag photo.jpg
```

Example:

```text
File
  Detected format: JPEG
  Dimensions: 4032 × 3024 pixels
  SHA-256: ...

Privacy context
  Categories found: location, capture time, device
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
