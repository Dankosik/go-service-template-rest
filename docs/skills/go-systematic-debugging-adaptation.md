# Адаптация `systematic-debugging` Под `go-service-template-rest`

## 1. Источник

- Репозиторий: `https://github.com/obra/superpowers`
- Зафиксированный коммит: `e4a2375cb705ca5800f0833528ce36a3faf9017a`
- Базовый skill: `.agents/skills/go-systematic-debugging/SKILL.md`
- Связанные материалы:
  - `root-cause-tracing.md`
  - `defense-in-depth.md`
  - `condition-based-waiting.md`

## 2. Что В Исходнике Ценно

Что переносится почти без изменений (концептуально):
- жесткий принцип `root cause before fix`;
- фазовый цикл `investigate -> analyze patterns -> hypothesis -> implement`;
- запрет на bundle из нескольких speculative fixes;
- обязательная проверка после фикса;
- акцент на flaky-тестах и condition-based waiting.

## 3. Что В Исходнике Слишком Generic Для Нашего Репо

1. Tooling примеры на Node/TypeScript (`npm test`, `setTimeout`, TS snippets).
2. Ссылки на внешние superpowers-skill dependencies (`superpowers:test-driven-development`, `superpowers:verification-before-completion`).
3. Нет привязки к нашему spec-first lifecycle (`Gate G2/G3/G4`, `Spec Freeze`, `Spec Clarification Request`, `Spec Reopen`).
4. Нет привязки к нашим quality-командам (`make test`, `make test-race`, `make openapi-check`, `make lint`).
5. Нет явного маппинга на архитектурные слои репозитория (`internal/app`, `internal/domain`, `internal/infra/*`, `cmd/service/main.go`).

## 4. Матрица Адаптации

### Keep (оставить)
- фазы дебага как дисциплину принятия решений;
- "one hypothesis, one experiment";
- отдельные anti-patterns и red flags;
- подход с supporting techniques (`root-cause tracing`, `defense-in-depth`, `condition-based waiting`).

### Adapt (переписать под стек)
- команды и примеры на Go/Makefile;
- проверка race/concurrency через `make test-race`/`go test -race`;
- error-chain диагностика через `errors.Is`/`errors.As`;
- фаза эскалации в spec-first процесс при semantic drift.

### Drop (исключить)
- зависимости от внешних superpowers-skills;
- TS-specific snippets;
- platform-specific советы, которые не соответствуют нашей структуре/CLI.

## 5. Что Уже Сделано В Репозитории

Создан новый runnable skill:
- `.agents/skills/go-systematic-debugging/SKILL.md`

Добавлены reference-файлы под Go:
- `.agents/skills/go-systematic-debugging/references/root-cause-tracing-go.md`
- `.agents/skills/go-systematic-debugging/references/defense-in-depth-go.md`
- `.agents/skills/go-systematic-debugging/references/condition-based-waiting-go.md`

Ключевые изменения относительно оригинала:
- интеграция со spec-first эскалацией (`Spec Clarification Request` / `Spec Reopen`);
- привязка к локальным командам из `docs/build-test-and-development-commands.md`;
- привязка к локальной архитектуре модулей;
- формализованный выходной формат debugging-отчета;
- `Context Intake` через локальные `docs/llm/*` файлы по trigger-модели.

## 6. Рекомендованные Дальнейшие Кастомизации

1. Ввести небольшой "debug evidence template" в `reviews/<feature-id>/` для одинакового формата RCA между сессиями.
2. Добавить обязательный пункт cleanup для временной диагностики (удаление debug-инструментации до финального handoff).
3. Добавить 2-3 example сценария под ваш частый дефект-профиль:
   - API boundary validation mismatch;
   - race/leak в goroutine lifecycle;
   - cache fallback inconsistency.
4. После обкатки на 3-5 реальных инцидентах откалибровать `Review Blockers` (если окажутся слишком жесткими или слишком мягкими).

## 7. Риски Интеграции

- Новый skill может частично пересекаться с `go-qa-review`, `go-reliability-review`, `go-concurrency-review`.
- Снижать риск нужно через strict handoff rule: debugging skill находит root cause и план минимального фикса, а domain-review skills подтверждают качество в своих доменах.

## 8. Итог

Адаптация `systematic-debugging` в этот репозиторий практична и low-effort, если использовать ее как process-skill для bugfix/incident ветки, а не как замену существующим spec/review ролям.
