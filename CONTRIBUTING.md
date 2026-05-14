# Contributing to Deal Sense

## Local setup

После клонирования включите git hooks из `.githooks/`:

```bash
git config core.hooksPath .githooks
```

Это однократная настройка для каждого clone. Hooks хранятся в репозитории и обновляются вместе с кодом.

### Что делают hooks

| Hook | Что проверяет | Блокирует commit |
|---|---|---|
| `pre-commit` | `gofmt` на staged `.go` файлах | да, если есть unformatted |
| `pre-commit` | `golangci-lint run ./...` (если установлен) | да, если есть issues |
| `pre-commit` | `gitleaks protect --staged` (если установлен) | да, если найден secret |
| `pre-commit` | `scripts/check-i18n.sh` на staged i18n-файлах | да, при 'ты'/`--`/`...` нарушениях |
| `commit-msg` | Conventional Commits формат | да, при mismatch |

Optional tools (golangci-lint, gitleaks) — soft-skip с install-hint'ом если не установлены. Это сделано чтобы первый день contributor'а не упёрся в дополнительные deps.

### Установка optional tools

```bash
brew install golangci-lint   # ~30 секунд lint, ловит баги до push
brew install gitleaks        # secrets scan, ловит API ключи в diff
```

## Conventional Commits

Все commit messages должны соответствовать [Conventional Commits 1.0](https://www.conventionalcommits.org/):

```
<type>(<scope>?)!?: <description>
```

Допустимые `type`: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `style`, `perf`, `build`, `ci`, `revert`.

`!` после scope означает breaking change.

### Примеры

| Хорошо | Плохо |
|---|---|
| `feat(auth): add OAuth2 flow` | `add oauth` |
| `fix: handle nil pointer in /healthz` | `fixed stuff` |
| `refactor(cmd)!: drop deprecated --legacy flag` | `cleanup` |
| `test(security): backfill marker uniqueness` | `added more tests` |

Auto-generated commits (`Merge ...`, `Revert ...`, `fixup!`, `squash!`) пропускаются валидатором.

## TDD дисциплина

Поведенческие изменения идут двумя коммитами:

1. `test(scope): RED — failing test for X` — запустить, увидеть RED.
2. `feat(scope): GREEN — implement X` — запустить, увидеть GREEN.

Покрытие уже написанного кода называется честно: `test: backfill coverage for X`. Не выдавать его за TDD.

Полные правила — в корневом `CLAUDE.md`.

## Verification перед push

```bash
cd backend && go test ./... -race -count=1 -timeout 180s
golangci-lint run ./...
```

Если оба зелёные — pushить можно.
