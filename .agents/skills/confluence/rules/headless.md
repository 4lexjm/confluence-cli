# Headless and CI Operation

In automated CI/CD runners, containerized environments, or background AI subagents,
interactive keychain prompts or manual login commands are unavailable.

`confluence` automatically synthesizes an active context and authenticates
using environment variables when present.

## Environment Variables

| Variable | Description | Example |
|---|---|---|
| `CONFLUENCE_HOST` | Confluence base URL | `https://myorg.atlassian.net` or `https://confluence.corp.internal` |
| `CONFLUENCE_TOKEN` | API Token (Cloud) or PAT (Data Center) | `my-api-token` |
| `CONFLUENCE_EMAIL` | Account email (Required for Cloud Basic Auth) | `ci-bot@myorg.com` |
| `CONFLUENCE_USER` | Username for DC Basic Auth | `srv_confluence` |
| `CONFLUENCE_SPACE` | Default space key | `ENG` |
| `CONFLUENCE_AUTH_METHOD` | Explicit auth mechanism: `basic`, `bearer`, `session-cookie`, `kerberos` | `bearer` |
| `CONFLUENCE_SESSION_COOKIE` | Raw `JSESSIONID` cookie string | `my-session-cookie` |
| `CONFLUENCE_HTTP_DEBUG` | Print full HTTP requests & responses to stderr | `1` |

## CI Pipeline Examples

### GitHub Actions (Cloud)

```yaml
jobs:
  publish-docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install confluence-cli
        run: go install github.com/atlassian/confluence-cli/cmd/confluence@latest
      - name: Publish Documentation
        env:
          CONFLUENCE_HOST: ${{ secrets.CONFLUENCE_HOST }}
          CONFLUENCE_EMAIL: ${{ secrets.CONFLUENCE_EMAIL }}
          CONFLUENCE_TOKEN: ${{ secrets.CONFLUENCE_API_TOKEN }}
          CONFLUENCE_SPACE: "DOCS"
        run: |
          confluence page create \
            --title "Build Report #${{ github.run_number }}" \
            --file ./dist/report.md
```

### Gitlab CI (Data Center PAT)

```yaml
docs_sync:
  stage: deploy
  script:
    - confluence page edit 45678 --file README.md --version-comment "CI Sync"
  variables:
    CONFLUENCE_HOST: "https://confluence.internal"
    CONFLUENCE_TOKEN: "$CONFLUENCE_PAT"
    CONFLUENCE_SPACE: "ARCH"
```

## Security Best Practices
- Never check credentials into git repositories.
- Use runner secrets or vaulted environment variables.
- Restrict service account tokens to minimal necessary space permissions.
