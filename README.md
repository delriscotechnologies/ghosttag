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
ghosttag /path/to/image.jpg
```

## What it does

- Detects JPEG or PNG from the image data and reports its dimensions.
- Calculates SHA-256 over the input bytes.
- Reads JPEG APP1 EXIF/XMP metadata and JPEG comments before the first image scan.
- Reads PNG `eXIf`, `tEXt`, and uncompressed `iTXt` metadata.
- Reports supported location, capture-time, device, authorship/copyright, and comment/description values.
- Groups those findings into privacy-relevant categories and adds a context note when three or more categories are present.

Location is currently derived from supported EXIF GPS coordinates. XMP support focuses on `CreateDate`, `DateTimeOriginal`, TIFF make/model, and Dublin Core creator, rights, and description values.

## Output

```text
ghosttag — image metadata report

File
  Name: sample.png
  Detected format: PNG
  Size: 558 bytes
  Dimensions: 1 × 1 pixels
  SHA-256: 234e2aaecf8145e20ee32d6066321474877b031031878bb16ea5261883235bf2

Privacy context
  Categories found (4): capture time, device, authorship, comments
```

## Scope and limits

- Supported target: Linux.
- Accepts one regular JPEG or PNG file per run.
- Rejects files reported as larger than 100 MiB before reading.
- Makes no network calls during inspection and does not write to the input file.
- Does not decode pixels, recognize visual subjects, remove metadata, or scan directories.
- This is a focused metadata inspector, not a complete JPEG, PNG, TIFF, EXIF, or XMP implementation.
- PNG `zTXt`, compressed `iTXt`, XMP GPS, and metadata placed after the first JPEG image scan are outside the current scope.
- Metadata may be missing, stale, malformed, or intentionally misleading.

See [SECURITY.md](SECURITY.md) for the security boundary.

## License

GHOSTTAG is available under the [MIT License](LICENSE).
