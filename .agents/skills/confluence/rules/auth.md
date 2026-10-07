# Authentication

`confluence auth` manages host credentials and connection methods across
Confluence Cloud and Confluence Data Center.

## Supported Methods

### 1. Cloud API Token (Recommended for Confluence Cloud)
Create an Atlassian API token at https://id.atlassian.net/manage-profile/security/api-tokens.

```bash
confluence auth login https://myorg.atlassian.net \
  --email user@example.com \
  --token <API_TOKEN> \
  --space ENG
```

### 2. Personal Access Token (PAT) (Recommended for Data Center >= 7.9)
Generate a PAT in Confluence: Profile -> Personal Access Tokens.

```bash
confluence auth login https://confluence.example.com \
  --token <PAT> \
  --space DEV
```

### 3. Browser SSO / Session Cookie (Data Center with SSO/SAML/Okta)
Captures the `JSESSIONID` cookie automatically using a local reverse proxy.

```bash
confluence auth login https://confluence.example.com --browser
```

### 4. Kerberos / SPNEGO (Data Center with Active Directory)
Uses active Kerberos ticket cache (`kinit`) without storing credentials:

```bash
confluence auth login https://confluence.example.com --auth-method kerberos
```

## Subcommands

- `confluence auth status`: Displays current host, account, token presence, and session state.
- `confluence auth doctor`: Runs active connectivity and permission checks against Confluence.
- `confluence auth session`: Inspects session cookie expiry and source.
- `confluence auth logout [host]`: Removes credentials from keychain and clears host config.
