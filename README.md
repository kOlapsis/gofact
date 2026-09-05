# gofact

> 🇫🇷 [Version française](README-FR.md)

Small Go binary that turns a **print-ready HTML invoice** into
**Factur-X**: a compliant PDF/A-3 with the CII EN 16931 XML embedded, and
optionally submits it to a French e-invoicing platform (PDP —
*plateforme de dématérialisation partenaire*).

Built for the invoicing of a French freelancer or small business (the French
e-invoicing reform), and also shipped as a **Claude Code plugin** with the
`creer-facture` skill.

## Pipeline

```
JSON  ──EN 16931 rules──▶  rejected if non-compliant (before producing anything)
      ──gofact──────────▶  factur-x.xml (CII EN 16931)
HTML  ──headless Chrome─▶  PDF          (identical to Ctrl+P output, backgrounds included)
PDF + XML ──pure Go─────▶  Factur-X     (PDF/A-3 + VERBATIM embedding of the XML)
Factur-X  ──self-check──▶  re-read      (structure + integrity of the embedded XML)
```

> **Verbatim embedding — important.** The XML is embedded **byte for byte**.
> An assembler that re-serializes it through its own model **loses extended
> fields**, in particular the PDP electronic routing addresses (BT-34/BT-49).
> gofact never touches the XML it has produced.

## System dependencies

**A browser, and that's it.**

- **Chrome, Edge, Brave or Chromium** — HTML → PDF rendering

Auto-detected on Linux, macOS and Windows (where **Edge** is preinstalled, so
nothing to do). Sandboxed packages — snap, flatpak — are skipped: they cannot
read the temporary files gofact hands them. To point at a specific executable:
`GOFACT_CHROME` or `-chrome`.

The PDF/A-3 assembly is done in Go (pdfcpu): no Ghostscript, no Java, no
first-run download. The sRGB ICC profile is embedded in the binary.

## Compliance

Two checks, at two different moments:

- **EN 16931 business rules**, applied on the model **before** producing
  anything. gofact refuses to issue an invoice it knows to be non-compliant,
  reporting the rule identifier (`BR-50`, `BR-CO-15`…) and the field to fix.
- **PDF self-check**, after writing: the file is re-read, the Factur-X
  structures are verified (`/AF`, `/OutputIntents`, XMP, `/EmbeddedFiles`) and
  the embedded XML is compared byte for byte with the one that was generated.

Deep PDF/A-3b compliance is verified by **veraPDF**, via Mustang, in
continuous integration — never on the user's machine:

```sh
GOFACT_ORACLE_HTML=facture.html go test -tags=ci ./internal/facturx
```

## Build

```sh
go build -o gofact .
go test ./...
```

## Configuration

**No identity and no secret is hard-coded.** The invoice issuer, the payment
IBAN and the PDP credentials come from the environment.

```sh
cp .env.example .env    # then fill in your own values
```

`gofact` loads, in order and without ever overwriting an already-defined
variable: the file passed to `-env`, otherwise `./.env`, then
`$XDG_CONFIG_HOME/gofact/.env` (`~/.config/gofact/.env` by default). The `.env`
is git-ignored — never commit it.

| Variable | Purpose |
| --- | --- |
| `GOFACT_SELLER_NAME` | Seller name. **Required** unless the JSON carries a `seller` block. |
| `GOFACT_SELLER_SIRET` / `GOFACT_SELLER_SIREN` | French legal identifier (BT-30). The SIREN is derived from the SIRET. |
| `GOFACT_SELLER_VAT_NUMBER` | Intra-community VAT number (BT-31). Empty under the 293 B franchise ⇒ BT-32 = SIREN. |
| `GOFACT_SELLER_EMAIL` | Seller e-mail. |
| `GOFACT_SELLER_ADDRESS` / `_POSTAL_CODE` / `_CITY` / `_COUNTRY` | Postal address (country defaults to `FR`). |
| `GOFACT_SELLER_ELECTRONIC_ADDRESS` / `_SCHEME` | Peppol routing BT-34. Empty ⇒ SIREN + scheme `0225`. |
| `GOFACT_PAYEE_IBAN` | Payment IBAN (BT-84). Empty ⇒ omitted from the invoice. |
| `GOFACT_VAT_EXEMPTION_MENTION` / `_CODE` | Default exemption mention and code (293 B). |
| `SUPERPDP_CLIENT_ID` / `SUPERPDP_CLIENT_SECRET` / `SUPERPDP_BASE` | PDP credentials (**secrets**), for `-send` only. |

With no seller configured, `gofact` **fails explicitly** rather than issuing an
incomplete invoice. Every value can still be overridden per invoice via the JSON.

## MCP server — talk to your AI to invoice

`gofact mcp` exposes invoicing as a **local MCP server (stdio)**: from Claude
Desktop, Claude Code, LM Studio or any MCP client, saying "make me an invoice
for ACME, 2 days at €600" is enough. The AI is the interface; gofact
guarantees the numbering (continuous legal sequence, locked and transactional
allocation), EN 16931 compliance and archiving — locally.

```sh
# Register in Claude Code, for example:
claude mcp add gofact -- /path/to/gofact mcp
```

Tools: `list_organizations`, `get_organization`, `init_organization`,
`update_organization`,
`search_client`, `get_invoice_template`, `preview_next_number`, `list_invoices`,
`create_invoice`, `send_invoice` (the only destructive tool — PDP submission,
explicit confirmation required), `get_invoice_status`. Prompt: `nouvelle-facture`.

An **organization** (issuing entity) = a self-contained directory: identity in
its `.env`, `numerotation.json` registry, `journal.ndjson` audit log, invoice
template frozen at first issuance, and the invoices themselves. CLI
management: `gofact org list | init | show`. No secret ever leaves a tool
response.

## Usage (CLI)

```sh
# Simplest form: -data and -out derived from the HTML file name
gofact -html "2026011 - Client.html"
#   → reads "2026011 - Client.json", writes "2026011 - Client.pdf"

# Explicit paths + XML dump for debugging
gofact -html f.html -data f.json -out f.pdf -xml f.xml

# Options
-env <path>       # explicit .env file
-validate=false   # skip the self-check of the produced PDF
-chrome <path>    # force the Chrome executable
-q                # quiet (errors only)
```

Exit code `0` if the Factur-X is generated (and compliant when `-validate`).

## Sending to a PDP (SuperPDP)

`gofact` submits the Factur-X PDF to SuperPDP (OAuth2 client credentials).

```sh
# generate then submit in one command
gofact -html f.html -send -poll
# submit an already-generated PDF
gofact send -pdf f.pdf -poll
```

> **The PDP requires that the invoice seller = the company of the authenticated
> account** (legal identifier BT-30) and an **electronic routing address**
> (BT-34/BT-49, scheme `0225`) for both the issuer and the recipient. See
> `electronic_address` below. `-poll` shows the lifecycle (`fr:200` submitted →
> `fr:201` issued → `fr:202` received).

## JSON format (sidecar)

All amounts are in **cents** (integers). The seller, the IBAN, the VAT regime
and the French legal mentions have **default values** (environment) — the JSON
only carries what varies.

```json
{
  "number": "2026011",
  "type": "invoice",
  "issue_date": "2026-06-29",
  "buyer_reference": "D2026004",
  "buyer": {
    "name": "ACME SAS",
    "siret": "55208131700015",
    "email": "compta@acme.example",
    "address": "1 rue de la Paix",
    "postal_code": "75002",
    "city": "Paris",
    "country_code": "FR"
  },
  "lines": [
    { "name": "Développement", "unit": "day", "quantity": "2.00",
      "unit_price_ht_cents": 60000, "amount_ht_cents": 120000 }
  ]
}
```

Optional fields: `due_date` (defaults to "on receipt" = issue date),
`delivery_date` (defaults to issue date), `currency` (defaults to EUR), `vat`
(defaults to 293 B exemption; set `{"exempt": false, "rate_pct": "20.00"}` for
standard VAT), `seller`, `iban`, `notes`.

**PDP routing**: `seller`/`buyer` accept `electronic_address` (BT-34/BT-49) and
`electronic_address_scheme` (e.g. `"0225"`). Required for PDP submission, which
routes on these addresses rather than on the postal address. Example (seller
override):

```json
"seller": {
  "name": "Studio Exemple", "siren": "123456789",
  "electronic_address": "123456789", "electronic_address_scheme": "0225"
}
```

## Installation

**Precompiled binary** (GitHub release, no toolchain required):

```sh
curl -fsSL https://raw.githubusercontent.com/kolapsis/gofact/main/install.sh | sh
gofact install -yes    # registers the MCP server in the detected clients
```

`gofact install` detects Claude Desktop, Claude Code, LM Studio and Cursor,
shows what it intends to write (dry-run by default), backs up every file before
modifying it and never overwrites a diverging entry without `-force`. Windows:
`install.ps1`.

**Package managers** — the cask and the manifest live in this repository, hence
the explicit URLs (Homebrew's short `brew tap` form requires a `homebrew-*`
repository name; Homebrew core and the main Scoop bucket both gate on a
popularity threshold gofact has not reached):

```sh
brew tap kOlapsis/gofact https://github.com/kOlapsis/gofact
brew install kOlapsis/gofact/gofact
```

```powershell
scoop bucket add gofact https://github.com/kOlapsis/gofact
scoop install gofact
```

**Claude Code plugin** — the repository is also a plugin: `creer-facture` skill
+ declared MCP server:

```
/plugin marketplace add kolapsis/gofact
/plugin install gofact@gofact
```

**From source**: `go build -o gofact .`

Organizations live in **local directories** (identity, registry, invoices) —
never versioned here: they contain real data. `GOFACT_INVOICES_DIR` is still
honored for compatibility with the historical skill.

## Known limitations

- Rendering relies on a Blink browser; sandboxed packages (snap, flatpak) are
  ignored because they cannot read outside `$HOME`.
- Single VAT rate. Multi-rate is not supported.
- Currency ≠ EUR: the VAT counter-value in EUR (BT-111) must be provided
  upstream (`VATEur` field of the model); not exposed in the JSON yet.
- Only one PDP implemented (SuperPDP).

## License

**GNU AGPL v3 or later** — see [LICENSE](LICENSE).

Copyright (C) 2026 Benjamin Touchard.

gofact is free software: you can redistribute it and/or modify it under the
terms of the GNU Affero General Public License as published by the Free
Software Foundation, either version 3 of the License, or any later version. It
is distributed in the hope that it will be useful, but **WITHOUT ANY
WARRANTY**; without even the implied warranty of merchantability or fitness
for a particular purpose.
