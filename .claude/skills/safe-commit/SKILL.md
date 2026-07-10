---
name: safe-commit
description: Scan staged (or about-to-be-staged) changes for secrets, tokens, API keys, and credentials before running git commit in this repo. Use this before EVERY git commit here — hey-bob touches Discord bot tokens, Upbit/Bithumb exchange API keys, and Claude API keys, any of which leaking is a real incident, not a lint warning.
---

# safe-commit

Pre-commit secret gate for the hey-bob repo. Run this before any `git commit`,
whether the user asked for the commit directly or an agent (developer/executor)
is committing as part of its own workflow.

## Steps

1. **See what's actually about to be committed.**
   - `git status` (never `-uall`) to see staged/unstaged/untracked files.
   - If nothing is staged yet but the user wants specific files committed, stage
     only those named files — never `git add -A` / `git add .` here, since that
     can silently pull in `.env`, DB files, or stray credential dumps sitting in
     the working tree.
   - `git diff --cached` to get the actual added/removed lines for staged files.

2. **Check filenames first (cheap, catches the worst cases).**
   Flag if any staged path matches:
   - `.env`, `.env.*` (but **not** `.env.example` / `.env.sample`)
   - `*.pem`, `*.key`, `id_rsa`, `id_ed25519`, `*.p12`, `*.pfx`
   - `*.sqlite`, `*.sqlite3`, `*.db` (already gitignored repo-wide, but double
     check nothing was force-added with `-f`)
   - anything under a path containing `secret`, `credential`, or `token` that
     isn't source code (e.g. a dumped JSON/txt file, not a `.go`/`.ts` file that
     merely defines a `Token` field)

3. **Grep the staged diff content for secret-shaped strings.**
   Run against `git diff --cached` (added lines only, i.e. lines starting with
   `+`, excluding the `+++` file headers):

   ```bash
   git diff --cached -- . ':(exclude)*.env.example' | grep -nE '^\+' | grep -vE '^\+\+\+' | grep -inE \
     -e '(api[_-]?key|api[_-]?secret|secret[_-]?key|access[_-]?key|access[_-]?token|client[_-]?secret|private[_-]?key|passwd|password)[[:space:]]*[:=][[:space:]]*["'"'"'][A-Za-z0-9+/_.=\-]{8,}["'"'"']' \
     -e 'AKIA[0-9A-Z]{16}' \
     -e 'sk-ant-[A-Za-z0-9_-]{20,}' \
     -e 'sk-[A-Za-z0-9]{20,}' \
     -e 'AIza[0-9A-Za-z_-]{35}' \
     -e 'xox[baprs]-[A-Za-z0-9-]{10,}' \
     -e '[MN][A-Za-z0-9_-]{23}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27,}' \
     -e 'discord(app)?\.com/api/webhooks/[0-9]+/[A-Za-z0-9_-]+' \
     -e '-----BEGIN ([A-Z ]+)?PRIVATE KEY-----' \
     -e 'eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}'
   ```

   These cover: generic `key=`/`token=`/`password=` assignments, AWS access
   keys, Anthropic (`sk-ant-`) and OpenAI-style (`sk-`) keys, Google API keys,
   Slack tokens, Discord bot tokens and webhook URLs, PEM private key blocks,
   and JWTs. Upbit/Bithumb access/secret keys don't have a fixed prefix, so
   they're caught by the generic `key=`/`secret=` assignment pattern — treat
   any `access_key`/`secret_key` literal (not a variable reference or
   `os.Getenv(...)` call) as a hit.

4. **If `gitleaks` is installed** (`which gitleaks`), also run
   `gitleaks protect --staged --no-banner` from the repo root and merge its
   findings with the manual grep — it catches patterns the list above misses.
   If it's not installed, don't stop to install it; the grep pass is the
   baseline and is mandatory either way.

5. **Triage matches — don't just flag every hit.**
   - Placeholder values (`YOUR_API_KEY_HERE`, `xxx`, `changeme`, `<token>`,
     empty string) are not real leaks — note them as fine.
   - A struct/type definition like `AccessKey string \`json:"access_key"\`` or
     `os.Getenv("DISCORD_TOKEN")` is not a leak — it's a reference, not a value.
   - A real hit is a literal secret-shaped value assigned or hardcoded where a
     reference/env-lookup should be.

6. **Gate the commit.**
   - If nothing real is found: say so in one line and proceed with the commit
     the user asked for.
   - If something real is found: **do not commit.** List the exact file(s) and
     line(s) hit, explain why each looks like a secret, and stop. Suggest the
     fix (move the value to `.env` / a secret store and reference it via
     `os.Getenv`, or unstage the file) but let the user decide — don't rewrite
     their code or scrub history on your own.
