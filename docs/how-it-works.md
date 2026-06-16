# How It Works

1. **URL Parsing** — Extracts OCP version and document structure from Red Hat documentation URLs
2. **Version Discovery** — Checks newer OCP versions to see if the same document exists
3. **URL Validation** — Verifies that suggested URLs are accessible (HTTP HEAD/GET requests)
4. **Anchor Validation** — When a URL contains a fragment (`#anchor`), the tool:
   - Fetches and parses the HTML page
   - Searches for the anchor ID in the page content
   - Reports when a page exists but the anchor is missing
   - Prevents false positives when content is reorganized between versions
5. **Smart Recommendations** — Only suggests versions where both the page AND anchor (if present) exist

## Anchor Validation Details

The tool validates anchors by checking for:
- HTML `id` attributes (e.g., `<h2 id="section-name">`)
- Legacy `<a name="anchor">` tags
- Red Hat docs style anchors (e.g., `#installing-operator_procedure-name`)

This prevents suggesting URLs where the documentation page exists but the specific section has moved or been renamed.

**Example:** If you have a URL pointing to 4.17 with anchor `#installing-sr-iov-operator_installing-sriov-operator`, and in 4.19 the SR-IOV content moved from the networking guide to the hardware_networks guide, the tool will detect that the anchor doesn't exist in the 4.19 networking guide and won't suggest it as an upgrade path.

**Performance Note:** Anchor validation requires fetching and parsing full HTML pages, which is slower than simple HEAD requests. For URLs without anchors, the tool uses fast HEAD requests. URLs with anchors will take longer to validate (typically 1-3 seconds per URL).

## Supported URL Formats

The tool supports Red Hat OpenShift Container Platform documentation URLs in the following formats:

- `https://docs.redhat.com/en/documentation/openshift_container_platform/{version}/html-single/{document}/index#{anchor}`
- `https://docs.redhat.com/en/documentation/openshift_container_platform/{version}/html/{document}/{page}#{anchor}`

Examples:
- Single-page HTML: `.../4.17/html-single/disconnected_environments/index#anchor`
- Multi-page HTML: `.../4.17/html/telco_ref_design_specs/telco-hub-ref-design-specs#anchor`
