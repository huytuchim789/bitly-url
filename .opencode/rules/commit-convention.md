# Commit Convention

## Format

```
<type>(<scope>): <description>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

## Types

| Type     | Usage                                      |
|----------|--------------------------------------------|
| `feat`   | New feature                                |
| `fix`    | Bug fix                                    |
| `refactor` | Code change without fix or feature       |
| `chore`  | Tooling, config, deps, CI                  |
| `docs`   | Documentation only                         |
| `style`  | Formatting, lint, whitespace (no logic)    |
| `test`   | Adding or fixing tests                     |
| `perf`   | Performance improvement                    |
| `ci`     | CI/CD pipeline changes                     |

## Scopes

| Scope      | Area                                    |
|------------|-----------------------------------------|
| `root`     | Root config, Makefile, docker-compose   |
| `server`   | Go/Gin backend                          |
| `client`   | Next.js frontend                        |
| `hooks`    | Git hooks (husky)                       |
| `ci`       | GitHub Actions workflows                |
| `docker`   | Dockerfiles, compose                    |
| `docs`     | README and documentation                |
| `opencode` | opencode rules, agents, skills          |

## Rules

1. **Atomic** — one commit = one logical change. Do NOT mix scopes.
2. **Imperative present tense** — "add feature", NOT "added" or "adds"
3. **Lowercase description** — no caps unless proper noun
4. **No trailing period** — description has no dot at end
5. **Body explains WHY** — the motivation, not the implementation detail
6. **Footer for breaking changes** — `BREAKING CHANGE: <description>`
7. **Max 72 chars** for the subject line

## Examples

```
feat(server): add health check endpoint

fix(client): handle null response in useUrls hook

refactor(server): extract validation to separate middleware

chore(root): add .gitattributes for cross-platform line endings

ci: add git-secrets scan to GitHub Actions

docs: update README with make install instructions

chore(opencode): add commit convention rules and agent
```
