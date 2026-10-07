# Comments

Comments provide inline and footer discussion on Confluence pages.

## Commands

### List comments on a page
```bash
confluence comment list 123456
confluence comment list 123456 --limit 50
```

### Add a comment
Markdown formatting is supported.

```bash
# Add top-level comment
confluence comment add 123456 \
  --body "Please review section 3: the schema needs clarification."

# Reply to an existing comment
confluence comment add 123456 \
  --parent 78910 \
  --body "Updated the schema diagram according to your feedback."

# Add comment from a file
confluence comment add 123456 --file ./review-notes.md
```

### Edit a comment
```bash
confluence comment edit 78910 --body "Correction: reviewed and approved."
```

### Delete a comment
```bash
confluence comment delete 78910 --confirm
```
