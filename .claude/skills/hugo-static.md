---
name: hugo-static
description: This skill focuses on managing Hugo static site generation, which is a popular open-source framework for building static websites. It includes creating, updating, and deleting Hugo site configurations, as well as handling content management and deployment processes.
---

## Purpose
Used when creating or modifying the logic for managing Hugo static sites. This includes defining the structure of the Hugo site, handling content updates, and ensuring that the site is correctly generated and deployed.

## Hugo site content structure
- content/: This directory contains the markdown files for the site's content, organized into sections and pages.
- layouts/: This directory contains the HTML templates for the site's layout and design.
- static/: This directory contains static assets such as images, CSS, and JavaScript files.
- config.toml: This file contains the configuration settings for the Hugo site, including site parameters, theme settings, and deployment configurations.


### Feature card linking
- Feature cards on the home page are defined in `hugo.toml` under `[[params.features]]`
- To make a card link to a documentation page, add a `url` field to the feature entry:
  ```toml
  [[params.features]]
    icon        = "🛡️"
    title       = "Policy Definitions"
    url         = "/docs/policy-definitions/"
  ```
- The home page template (`themes/policy-operator-theme/layouts/index.html`) checks for `.url`:
  - Present → renders the card as `<a class="feature-card feature-card-link">` (clickable)
  - Absent  → renders the card as a plain `<div class="feature-card">` (no link)
- The `feature-card-link` CSS class is already defined in `style.css` and handles hover styles

### Content management
- CreateContent: Create new markdown files in the content/ directory, ensuring that they are properly formatted and organized according to the site's structure.
- UpdateContent: Update existing markdown files in the content

### Content
- PolicyDefinitions/PolicyAssignment/PolicyInitiave/PolicyExemption:
    - PolicyDefintion api schema : the path is /api/v1alpha1/azurepolicydefinition_types.go
    - PolicyAssignment api schema : the path is /api/v1alpha1/azurepolicyassignment_types.go
    - PolicyInitiave api schema : the path is /api/v1alpha1/azurepolicyinitiative_types.go
    - PolicyExemption api schema : the path is /api/v1alpha1/azurepolicyexemption_types.go
    - The policy definition yaml manifest schema shold be refered from the api schema
    - Documentation page: policy-operator/content/docs/policy-definitions.md
    - UI style: Upbound/Crossplane marketplace style (https://marketplace.upbound.io)
      - Tabbed container: "API Documentation" tab + "Examples (N)" tab
      - Full nested field tree: apiVersion → kind → metadata → spec (all fields) → status
      - Object fields (spec, metadata, policyRule) expand inline to show child field rows in a bordered sub-tree
      - Leaf fields expand to show description, allowed enum values, default, and mutual-exclusion notes
      - Field rows: purple square +/- toggle | blue underlined field name | spacer | [required pill] [type badge]
    - Shortcodes used (all in policy-operator/themes/policy-operator-theme/layouts/shortcodes/):
      - `api-schema` : outer tabbed wrapper; params: kind, version, examples (count), status ("true" to show Status tab)
      - `api-field`  : single field row; params: name, type, required, desc, children, default, mutual, enum
                       - set children="true" for Object/Array fields whose .Inner contains nested api-field shortcodes
                       - omit children for leaf fields; .Inner is markdown description/examples
      - `api-examples` : holds example YAML content; JS moves it into the Examples tab automatically
      - `api-status`   : holds status field rows (api-field shortcodes); JS moves it into the Status tab automatically
                         - place after {{< /api-examples >}} (or after {{< /api-schema >}} if no examples)
                         - source node class: `.api-status-src` (hidden via CSS, same pattern as `.api-examples-src`)
    - CSS: .api-schema, .api-field, .api-field__children blocks in
           policy-operator/themes/policy-operator-theme/static/css/style.css (section 17)
    - Inline field examples: each api-field in the API Documentation tab can include a YAML
      code block in its .Inner content (markdown fenced ```yaml ... ```) — rendered and
      copy-buttoned automatically; already done for all fields in policy-definitions.md
    - JS:  tab switching + delegated accordion toggle + copy buttons in
           policy-operator/themes/policy-operator-theme/static/js/main.js
      - Copy button pattern: initCopyButtons(root) wraps every <pre> in .code-wrapper and
        appends a .code-copy <button>; a single delegated listener on `document` handles
        all clicks (works for dynamically added content such as the examples panel)
      - After moving api-examples-src innerHTML into the examples panel, call
        initCopyButtons(exPanel) so code blocks in the Examples tab also get copy buttons
    - Examples in the Examples tab (4 total):
        - Audit storage accounts without HTTPS (inline policyRule + parameters)
        - Deny untagged resources at management group scope
        - Import large policy from raw JSON using policyRuleJson
        - Audit VMs with public IPs (no parameters)

### Dark mode
- Toggle button (moon/sun SVG icons) is rendered in `layouts/partials/header.html`, placed before `.navbar-toggle`
- Icon visibility: `.icon-moon` shown by default; `[data-theme="dark"] .icon-sun` shown in dark mode
- FOUC prevention: inline `<script>` in `layouts/partials/head.html` reads `localStorage.getItem('theme')` and sets `data-theme` on `<html>` before first paint
- JS toggle logic in `static/js/main.js` — reads/writes `data-theme` on `document.documentElement` and persists to `localStorage`
- Dark palette applied via `[data-theme="dark"]` selector overriding CSS variables:
  - `--bg: #0f172a`, `--bg-alt: #1e293b`, `--border: #334155`
  - `--text: #cbd5e1`, `--text-muted: #94a3b8`
  - `--primary: #38bdf8`, `--primary-light: rgba(56,189,248,.12)`
- Heading/strong colours not covered by variables are overridden with explicit `[data-theme="dark"] h1…h6` rules (target colour `#f1f5f9`)
- API schema dark overrides use a scoped `:root` rule inside `[data-theme="dark"]` to redefine `--api-accent`, `--api-accent-bg`, `--api-border`, `--api-row-sep`
- All dark mode CSS is in section 20 at the bottom of `static/css/style.css`

### Dog caricature decorations
- Two SVG caricature files: `static/img/pug-caricature.svg` (left) and `static/img/labrador-caricature.svg` (right)
- Rendered in `layouts/_default/baseof.html` as fixed-position `<div class="dog-decor dog-pug|dog-labrador">` with `aria-hidden="true"`, placed just before `</body>`
- CSS in `static/css/style.css` (Dog decorations section):
  - `position: fixed; top: 50%; transform: translateY(-50%)`
  - `.dog-pug { left: -150px }` — only right half of pug face visible
  - `.dog-labrador { right: -150px }` — only left half of lab face visible
  - Light mode: `opacity: 0.09; mix-blend-mode: multiply`
  - Dark mode: `opacity: 0.13; mix-blend-mode: screen` (set in the dark mode section)
- Pug design: warm fawn `#E8C88A` base, dark brown `#3D2010` mask around eyes/muzzle, large round eyes with catchlights, flat wide nose, open mouth with pink tongue, freckle dots
- Labrador design: cream-gold `#EDD9A0` base, long floppy ears `#C8A850`, almond warm brown eyes, large black nose, closed gentle mouth

### Sidebar page ordering
- Controlled by `weight` front-matter in each content file under `content/docs/`
- Current order: Introduction (1) → Getting Started (2) → API Reference (3) → Policy Definitions (10) → Policy Initiatives (20) → Policy Assignments (30) → Policy Exemptions (40)
- Sidebar template (`layouts/partials/doc-sidebar.html`) uses `.Pages.ByWeight` to sort

### Previous / Next navigation
- Rendered in `layouts/_default/single.html` as `<div class="doc-footer">` (NOT `<footer>` — using `<footer>` causes the site-wide `footer { background: var(--dark) }` rule to apply)
- Cards are compact: `max-width: 36%`, padding `.5rem .875rem`, background `var(--bg)` explicitly set