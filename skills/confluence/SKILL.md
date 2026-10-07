---
name: confluence
version: 0.1.0
description: Operate Confluence Cloud and Data Center spaces, pages, blog posts, comments, attachments, and CQL search with confluence. Use when a request names Confluence or confluence-cli; confirm the instance before platform-specific commands.
metadata:
  short-description: Confluence CLI for pages, spaces, search, and knowledge management
  compatibility: claude-code, codex-cli, gemini-cli, cursor
---

# Confluence CLI (`confluence`)

Use `confluence` for Confluence Cloud and Confluence Data Center / Server operations.
Prefer structured `--json`, `--yaml`, `--jq`, or `--template` output for automation.

## Start with the requested operation

Run `confluence --version` only when installation or compatibility is uncertain. Run
`confluence auth status` when the command needs authentication or the active instance
and context are unknown. Do not make setup a prerequisite for an already
configured read-only request.

Cloud and Data Center have different base URL patterns and authentication conventions.
Determine the deployment from the active context or environment variables.

Read [setup and common workflows](rules/guide.md) when installing, authenticating,
creating a context, or learning the main command patterns. For config-free CI,
containers, or background agents, read [headless authentication](rules/headless.md).

## Route to the command reference

Read only the reference for the requested command group:

- [pages](rules/page.md) — Create, view, edit, search, move, and copy pages
- [spaces](rules/space.md) — List, view, create, and delete spaces
- [comments](rules/comment.md) — View and post comments
- [attachments](rules/attachment.md) — List, upload, and download files
- [search & CQL](rules/search.md) — Query content via text or CQL syntax
- [authentication](rules/auth.md) — Token, session cookie, and Kerberos management
- [contexts](rules/context.md) — Manage multi-instance profiles and default spaces
- [MCP server](rules/mcp.md) — Model Context Protocol stdio server for agents
- [raw API](rules/other.md) — Direct REST endpoint access via `confluence api`

## Mutation and safety boundaries

- Listing, viewing, searching, and status checks are read-only.
- Page creation, editing, deleting, space deletion, comment posting, and attachment
  upload modify Confluence; perform them only when requested.
- Confluence page editing increments the page version. When updating a page, provide
  an informative `--version-comment` so humans and other agents understand the change.
- Never place API tokens or passwords directly in command arguments that could enter
  logs. Use environment variables (`CONFLUENCE_TOKEN`) or the OS keychain.

## References

- [page](rules/page.md) — Work with Confluence pages
- [space](rules/space.md) — Work with Confluence spaces
- [comment](rules/comment.md) — Manage page comments
- [attachment](rules/attachment.md) — Manage file attachments
- [search](rules/search.md) — Search content using text and CQL
- [auth](rules/auth.md) — Manage Confluence authentication credentials
- [context](rules/context.md) — Manage Confluence CLI contexts
- [mcp](rules/mcp.md) — Model Context Protocol server for agents
- [other](rules/other.md) — Raw API escape hatch
