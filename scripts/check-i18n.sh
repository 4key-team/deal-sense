#!/usr/bin/env bash
#
# Lints user-facing RU strings for style violations:
#   - 'ты'-form ("informal you") — CLAUDE.md mandates "Вы".
#   - ASCII '--' where typographic em-dash '—' is expected.
#   - ASCII '...' where ellipsis '…' is expected.
#
# Scope is narrow on purpose: only the files where strings end up on the
# user's screen. Adversarial test fixtures, memory/, .claude/, and code
# comments are exempt — they reference patterns adversarially or describe
# behaviour, not present copy to users.
#
# Exit 0 if clean, 1 if any violation is found.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

# User-facing RU string scopes. Add new files here when they become a
# translation target.
files=(
  frontend/src/i18n/ru.ts
  backend/internal/adapter/telegram/messages.go
)

fail=0

# 1. ты-form (informal "you") — RU must use "Вы" per CLAUDE.md.
ty_pattern='\b(ты|тебя|тебе|тобой|твой|твоя|твоё|твои|твоих|твоему|твоего)\b'
for f in "${files[@]}"; do
  [ -e "$f" ] || continue
  if hits=$(grep -nE "$ty_pattern" "$f" 2>/dev/null); then
    echo "✘ $f: 'ты'-форма в user-facing RU строке"
    echo "$hits" | sed 's/^/    /'
    fail=1
  fi
done

# 2. ASCII '--' inside a Russian-containing literal — should be '—' (em-dash).
for f in "${files[@]}"; do
  [ -e "$f" ] || continue
  if hits=$(grep -nE '"[^"]*[а-яА-Я][^"]* -- ' "$f" 2>/dev/null); then
    echo "✘ $f: ASCII '--' вместо тире '—'"
    echo "$hits" | sed 's/^/    /'
    fail=1
  fi
done

# 3. ASCII '...' inside a Russian-containing literal — should be '…'.
for f in "${files[@]}"; do
  [ -e "$f" ] || continue
  if hits=$(grep -nE '"[^"]*[а-яА-Я][^"]*\.\.\.' "$f" 2>/dev/null); then
    echo "✘ $f: ASCII '...' вместо многоточия '…'"
    echo "$hits" | sed 's/^/    /'
    fail=1
  fi
done

if [ $fail -eq 0 ]; then
  echo "✓ i18n check passed"
fi
exit $fail
