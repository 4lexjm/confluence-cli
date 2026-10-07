# Contexts

Contexts bind a target Confluence host with an optional default space.
They allow switching between different environments, accounts, or teams.

## Commands

### List contexts
```bash
confluence context list
# or alias
confluence context ls
```

### Create a context
```bash
confluence context create prod-cloud \
  --host https://myorg.atlassian.net \
  --space ENG \
  --set-active
```

### Switch active context
```bash
confluence context use staging-dc
```

### View active or named context
```bash
confluence context view
confluence context view prod-cloud
```

### Delete a context
```bash
confluence context delete old-ctx
```

### Command override flag
You can override the active context on any command using `-c` / `--context`:

```bash
confluence page list -c staging-dc
```
