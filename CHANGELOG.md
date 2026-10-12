# Changelog

All notable changes to this project will be documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-10-07
### Added
- Initial release of `confluence` CLI for Confluence Cloud and Confluence Data Center.
- Enterprise authentication support:
  - Cloud API tokens (email + API token).
  - Data Center Personal Access Tokens (PAT).
  - Session cookie capture via browser SSO reverse proxy (`--browser`).
  - Kerberos/SPNEGO negotiation via ticket cache (`kinit`) without stored credentials.
  - Headless / CI configuration via environment variables.
- Content & Page management:
  - List, view, create, edit, move, copy, delete, and view version history.
  - Bidirectional Markdown ↔ Confluence storage XHTML format conversion.
  - View pages in clean Markdown, raw storage XHTML, or view HTML.
  - Page labeling and discussion comments.
- Space management: list, view, create, and delete spaces.
- Full-text search and advanced Confluence Query Language (CQL) queries (`--cql`).
- Built-in Model Context Protocol (MCP) server exposing tools for AI agents (`confluence mcp serve`).
- Agent Skills specification (`skills/confluence/SKILL.md`) with comprehensive rules and mirrors.
- Direct REST API escape hatch (`confluence api <path>`).
- Output formatting via `--json`, `--yaml`, and `--jq`.

[Unreleased]: https://github.com/4lexjm/confluence-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/4lexjm/confluence-cli/releases/tag/v0.1.0
