# Security policy

GHOSTTAG treats image bytes, metadata, paths, and displayed strings as untrusted input.

The current implementation:

- reads one regular local file and does not modify it;
- checks the file's reported size and rejects inputs already larger than 100 MiB;
- uses Go's JPEG/PNG image configuration decoders for format detection and dimensions;
- bounds each parsed EXIF directory to 256 entries and individual referenced EXIF values to 1 MiB;
- accepts EXIF GPS locations only with rational DMS values and valid `N/S` and `E/W` direction references;
- limits each displayed metadata field to 32 distinct values;
- strips terminal control and Unicode format characters from displayed text and truncates long values;
- performs no network requests during inspection.

GHOSTTAG is deliberately a focused metadata inspector rather than a complete image validator. It does not parse every legal EXIF/XMP field, PNG text representation, or JPEG metadata placement. Malformed or hostile files may still expose parser defects, and metadata should not be treated as trustworthy simply because it was extracted successfully.

Report genuine security issues privately through the repository's GitHub Security Advisory interface. Do not place sensitive images, personal metadata, credentials, or exploit files in public issues.
