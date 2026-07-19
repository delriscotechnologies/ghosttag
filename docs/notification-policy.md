# ghosttag notification policy

## Purpose

`ghosttag` reports factual metadata and helps a person notice when several kinds of contextual information appear together. It intentionally avoids `HIGH`, `MEDIUM`, and `LOW` labels because the image file alone cannot establish likelihood, impact, audience, purpose, or an individual's circumstances.

This policy controls wording in the CLI. It is not a scientific risk model.

## Normalized categories

| Category | Included facts | Examples |
| --- | --- | --- |
| Location | Supported, validated GPS coordinate pairs | EXIF GPS latitude and longitude, XMP GPS coordinates from the same RDF description |
| Capture time | Recognized original or creation timestamps | EXIF DateTimeOriginal, XMP CreateDate |
| Device | Camera or image-capture make and model | EXIF Make and Model, XMP TIFF Make and Model |
| Authorship | Author, artist, owner, or copyright values | EXIF Artist, namespace-aware XMP creator, PNG Author |
| Comments | Free-form comments, captions, or descriptions | JPEG COM, EXIF UserComment, namespace-aware XMP description, PNG Comment |

Unknown XMP namespaces and malformed capture-time values do not contribute to category counts. Software and orientation remain visible in the report but are not included in this count. Technical tags that are not normalized by this version do not affect the policy.

The count is based on categories rather than tags. For example, five device tags still contribute one `device` category.

## Output behavior

| Categories found | CLI behavior |
| --- | --- |
| 0 | State that no supported privacy-relevant categories were found and clarify that this does not prove anonymity. |
| 1–2 | List the categories and facts without an elevated notice. |
| 3–5 | List the categories and add a gentle combined-context note. |

The combined-context note says that the details can reveal more context together and asks the user to consider the intended recipient or platform. It does not say that the file is dangerous or that harm is likely.

## Relationship to NIST guidance

The [NIST Privacy Risk Assessment Methodology](https://www.nist.gov/itl/applied-cybersecurity/privacy-engineering/resources) applies the risk model from NISTIR 8062 to help organizations analyze and prioritize privacy risks. The [NIST Privacy Framework](https://www.nist.gov/privacy-framework/getting-started-0) emphasizes potential problems for individuals, data actions, and context.

Those ideas support two design choices in `ghosttag`:

1. report the data that is present before interpreting it; and
2. avoid treating a tag count as a complete privacy-risk calculation.

The `3+` threshold is a `ghosttag` product notification policy. NIST does not define this threshold, and `ghosttag` does not present it as a NIST score or standard.

## Known boundaries

The policy cannot determine:

- whether metadata is true;
- whether a recipient already knows the information;
- whether disclosure is expected or permitted;
- the likelihood of a problematic use;
- the potential impact on a particular person; or
- legal or regulatory compliance.

Users should interpret the report in the context of the actual sharing decision.
