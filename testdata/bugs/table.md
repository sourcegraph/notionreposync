The subject line should be concise and easy to visually scan in a list of commits, giving context around what code has changed.

1. Prefix the subject with the primary area of code that was affected (e.g. `web:`, `cmd/searcher:`).
2. Limit the subject line to 50 characters.
3. Do not end the subject line with punctuation.
4. Use the [imperative mood](https://chris.beams.io/posts/git-commit/#imperative) in the subject line.

    | Prefer | Instead of |
    |--------|------------|
    | Fix bug in XYZ | Fixed a bug in XYZ |
    | Change behavior of X | Changing behavior of X |

Example:

> cmd/searcher: Add scaffolding for structural search

