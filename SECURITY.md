# Security Policy

## Supported Version

Security fixes are applied to the current `main` branch. No older release line is maintained yet.

## Reporting a Vulnerability

For a sensitive report, use GitHub's **Report a vulnerability** option in the repository's **Security** tab when it is available. If private vulnerability reporting is unavailable, contact the repository owner through the GitHub profile and request a private channel before sharing exploit details.

Do not publish sensitive proof-of-concept files, personal metadata, credentials, or exploit details in a public issue.

For non-sensitive hardening suggestions, open a regular GitHub issue with the smallest reproducible example possible.

## Security Boundary

`ghosttag` treats image metadata, file names, paths, and parser diagnostics as untrusted input. The inspector:

- opens only regular files and rejects symbolic-link inputs;
- uses atomic no-follow opening on Linux and verifies opened-file identity on other supported platforms;
- rejects files larger than 100 MiB and rejects common concurrent changes detected through size or modification-time differences;
- validates the JPEG and PNG container structure needed to locate supported metadata and dimensions;
- bounds JPEG marker traversal, validates complete start-of-frame headers, and resumes marker parsing after scan data;
- requires a first, unique PNG `IHDR`, validates dimensions and critical-chunk CRCs, ignores ancillary chunks with invalid CRCs, and rejects data after `IEND`;
- limits PNG chunk counts, metadata chunk sizes, and decompression of compressed PNG text;
- limits XMP nesting depth and token count and accepts supported properties only from recognized namespaces;
- keeps XMP GPS latitude and longitude values within the same RDF description instead of combining unrelated blocks;
- limits normalized values independently for each metadata field, locations, and parser warnings;
- validates capture-time formats and GPS direction and degree, minute, and second components before classification;
- retains distinct source containers when equivalent values occur in multiple containers; and
- neutralizes terminal control and Unicode format characters in reported values and CLI diagnostics.

The parser is not a complete JPEG or PNG decoder. The tool is intentionally read-only and offline during inspection. It does not:

- modify or remove metadata;
- upload images or reports;
- decode image pixels or inspect visual subjects;
- scan directories recursively;
- treat unknown XMP namespaces as standard metadata properties; or
- claim that an image is anonymous when no supported metadata is found.

Malformed files may still expose implementation defects. A concurrent writer may also attempt changes that preserve observable file attributes. Inspect an unchanged copy with minimum permissions and operating-system resource limits, and do not rely on `ghosttag` as the sole control for handling hostile files.
