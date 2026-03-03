# Адаптация `verification-before-completion` Под `go-service-template-rest`

## 1. Источник

- Репозиторий: `https://github.com/obra/superpowers`
- Коммит: `e4a2375cb705ca5800f0833528ce36a3faf9017a`
- Базовый файл: `skills/verification-before-completion/SKILL.md`

## 2. Цель Интеграции

Добавить process-guardrail, который запрещает позитивные claim'ы (`fixed`, `passing`, `ready`) без свежей верификации командами, совместимыми с текущим spec-first workflow и quality gates.

## 3. Что В Оригинале Хорошо

- strong principle: evidence before assertions;
- простой gate-function перед любым completion claim;
- понятные анти-паттерны (extrapolation, stale evidence, partial checks).

## 4. Что Требовало Адаптации

1. Tone/формулировки слишком персонализированы и не соответствуют стилю репозитория.
2. Нет маппинга на локальные команды (`make test`, `make openapi-check`, `make test-race`, `make lint`, `make build`).
3. Нет привязки к `Gate G3/G4` и `Spec Freeze` semantics.
4. Нет встроенной интеграции с текущими skill-ролями (`go-coder`, `go-qa-tester`, `go-systematic-debugging`).

## 5. Внедренный Вариант

Создан новый runnable skill:
- `skills/go-verification-before-completion/SKILL.md`

Добавлен reference:
- `skills/go-verification-before-completion/references/claim-proof-matrix.md`

Интеграционные hook’и внесены в:
- `skills/go-coder/SKILL.md`
- `skills/go-qa-tester/SKILL.md`
- `skills/go-systematic-debugging/SKILL.md`

## 6. Как Skill Встраивается В Workflow

### Где вызывать
- перед любым утверждением "готово", "фикс подтвержден", "checks green", "ready for G3/G4";
- перед merge/PR readiness statements;
- после bugfix в `go-systematic-debugging`.

### Что он не делает
- не заменяет `go-systematic-debugging`;
- не заменяет domain-review skills;
- не требует full CI для каждого локального micro-claim (используется smallest-sufficient proof).

## 7. Практические Правила Эксплуатации

1. Scope claim должен быть явно обозначен (targeted/package/repo/gate).
2. Scope проверки должен быть не уже scope claim.
3. При неполной проверке conclusion должен быть `not verified`.
4. Gate-claim (`G3/G4`) запрещен без покрытия соответствующих условий, а не только тестов.

## 8. Риски И Смягчение

### Риск: излишняя бюрократия
Смягчение:
- policy `smallest sufficient command set`;
- explicit distinction between focused and repo-wide claims.

### Риск: дублирование с `go-coder`/`go-qa-tester`
Смягчение:
- `go-verification-before-completion` действует как финальный gate на формулировку claim,
  а не как замена implementation/test responsibilities.

## 9. Рекомендуемый Пилот

- Пилот на 1 неделю для всех bugfix и behavior-changing задач.
- Оценить:
  - число ложных "готово" сигналов,
  - число возвратов из-за недопроверки,
  - дополнительную стоимость по времени.
- По итогам: откалибровать claim->proof mapping для типовых задач вашей команды.
