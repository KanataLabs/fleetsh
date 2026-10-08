# Documentation domain and fleet patching implementation plan

> Execute the approved request in this session. Keep public mutations under the KanataLabs Publisher App identity.

**Goal:** Make English the default, use the GitHub Pages default URL, link fleetsh from the corporate product list, and document fleet-wide apt patching.

**Architecture:** Keep paired /en/ and /zh/ Jekyll routes. Keep https://kanatalabs.github.io/fleetsh/ with its /fleetsh base path and no CNAME. The corporate Astro site lists the product and links directly to its GitHub documentation URL. Documentation publishing uses an App token when configured, with a local Publisher utility fallback.

**Tech Stack:** Jekyll/GitHub Pages, Astro, PowerShell, Go CLI.

## Tasks

1. Update docs/_config.yml without adding a CNAME. Default docs/index.md and unlocalized routes to English. Update README links and internal/cli/cli.go.
2. Add paired docs/en/cases/ and docs/zh/cases/ index and patch-all-vps pages. Link cases from navigation, overviews and getting-started pages. Explain apt versus apt-get, selection, known-host setup, sudo, concurrency, JSON and reboot.
3. Publish through the organization App. Remove GITHUB_TOKEN-authenticated branch mutations from .github/workflows/docs.yml. Add scripts/publish-docs.ps1 for local App publication when the CI App secret is unavailable.
4. Add a fleetsh product card and Products navigation to the corporate site. Do not add redirects from corporate /fleetsh paths back to the default URL; GitHub domain inheritance would cause a loop.
5. Validate paired routes and internal links, run relevant CLI tests and build the corporate site. Push each repository with Kanata Labs commit identity; report any ruleset rejection without switching to personal credentials.
6. Verify the published case and English root redirect. Document the platform-provided 301 from the GitHub default address to the organization custom domain; changing that inheritance requires a separate hosting/domain change.

## Current deployment prerequisites

The fleetsh subdomain has no DNS record. The current Wrangler OAuth scope provides zone read access but no DNS write access. The corporate repository ruleset restricts creation and updates and has no Publisher App bypass. The App installation currently grants Contents, Pull requests and Workflows write, but does not grant Pages or Actions-secret write. No CI App private-key secret is configured.

Keep prepared changes reviewable and complete independent work before requesting any required external configuration.

User decision: keep the GitHub default address and place product links on the corporate website. Do not create a custom documentation domain or migrate the corporate site.
