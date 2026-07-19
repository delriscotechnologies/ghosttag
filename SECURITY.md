# Security Policy

## Supported Version

Security fixes are applied to the current `main` branch. No older release line is maintained yet.

## Reporting a Vulnerability

For a sensitive report, use GitHub's **Report a vulnerability** option in the repository's **Security** tab when it is available. If private vulnerability reporting is unavailable, contact the repository owner through the GitHub profile and request a private channel before sharing exploit details.

Do not publish sensitive proof-of-concept files, personal metadata, credentials, or exploit details in a public issue.

For non-sensitive hardening suggestions, open a regular GitHub issue with the smallest reproducible example possible.

## Security Boundary

`ghosttag` treats image metadata as untrusted input. The inspector:

- opens only regular files and rejects symbolic links and Linux special files;
- rejects files larger than 100 MiB;
- validates JPEG and PNG container structure and dimensions;
- limits PNG chunk counts and metadata chunk sizes;
- limits decompression of compressed PNG text;
- limits XMP nesting depth and token count;
- limits normalized metadata values and parser warnings;
- rejects non-finite and out-of-range GPS coordinates; and
- neutralizes terminal control and Unicode format characters before output.

The tool is intentionally read-only and offline during inspection. It does not:

- modify or remove metadata;
- upload images or reports;
- inspect image pixels or visual subjects;
- scan directories recursively; or
- claim that an image is anonymous when no supported metadata is found.

The project-local Go bootstrap verifies the official archive checksum and recreates the ignored `.tools/go` directory from that archive instead of executing an existing unverified toolchain.

Malformed files may still expose implementation defects. Run `ghosttag` with minimum permissions and operating-system resource limits, and do not rely on it as the sole control for handling hostile files.
