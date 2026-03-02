# Skill Spec: `go-qa-review` (Domain-Scoped Review)

## 1. Назначение

`go-qa-review` — экспертный review-skill по качеству тестов в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса для Go-сервисов.

Ценность skill:
- подтверждает, что тестовая реализация соответствует утвержденному `70-test-plan.md`;
- выявляет пробелы покрытия, которые могут пропустить регрессии в критичных сценариях;
- удерживает test-suite в стабильном, детерминированном и поддерживаемом состоянии до merge.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за QA-review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-qa-review` принимает решения по:
- полноте тестового покрытия против утвержденной спецификации:
  - соответствие матрице из `70-test-plan.md`;
  - покрытие обязательных сценариев `unit/integration/contract`;
- качеству тест-дизайна и проверок:
  - корректность assert-ов и проверяемых ожиданий;
  - отсутствие "пустых" тестов, которые не валидируют важное поведение;
- устойчивости и воспроизводимости test-suite:
  - детерминизм, изоляция, контроль времени/рандома/внешних зависимостей;
  - признаки flaky-тестов и ложной уверенности в качестве;
- traceability тестов к spec-обязательствам:
  - связь с `70-test-plan.md`;
  - связь с требованиями из `15-domain-invariants-and-acceptance.md` и `55-reliability-and-resilience.md` через тест-план;
- readiness тестового слоя к `Gate G4`:
  - отсутствие критичных test-gap в измененном функционале;
  - соответствие проектным quality-check ожиданиям для тестов.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-qa-review`:
- Phase 4 review с фокусом на проверку тестового кода и его соответствия `70-test-plan.md`.

Обязательная ответственность в каждом проходе:
- проверять, что реализованные тесты покрывают утвержденные критичные сценарии;
- фиксировать findings только в QA/test quality домене;
- ссылаться на конкретный `file:line` и соответствующий spec-source (`70`, и при необходимости `15/55/30/40/60/90`);
- давать practical corrective actions вместо абстрактных замечаний;
- не редактировать spec-файлы во время code review;
- при выявлении spec-level рассогласования инициировать `Spec Reopen` в `reviews/<feature-id>/code-review-log.md`.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Coverage Conformance`:
  - реализованы ли обязательные сценарии из `70-test-plan.md`;
  - нет ли критичных пропусков в `unit/integration/contract` слоях;
- `Critical Scenario Verification`:
  - проверяются ли сценарии, влияющие на инварианты и fail-path контракты (в объеме, зафиксированном в тест-плане);
  - нет ли перекоса только в happy-path при обязательных негативных сценариях;
- `Assertion Quality`:
  - проверяют ли тесты поведение, а не только факт отсутствия panic/error;
  - нет ли слабых assert-ов, скрывающих регрессии;
- `Stability And Determinism`:
  - нет ли гонок по времени, shared state leakage и хрупких зависимостей окружения;
  - контролируются ли источники nondeterminism;
- `Maintainability Of Test Suite`:
  - тесты читаемы и локализуют причину падения;
  - test helpers/fixtures не маскируют важные проверки.

## 5. Границы Экспертизы (Out Of Scope)

`go-qa-review` не подменяет соседние review-роли:
- не выполняет архитектурный review как primary-домен (`go-design-review`);
- не выполняет idiomatic/language-style review как primary-домен (`go-idiomatic-review`, `go-language-simplifier-review`);
- не выполняет глубокую проверку доменной корректности как primary-домен (`go-domain-invariant-review`);
- не выполняет performance/concurrency/security/reliability/db-cache аудит как primary-домен (`go-performance-review`, `go-concurrency-review`, `go-security-review`, `go-reliability-review`, `go-db-cache-review`).

Также вне scope:
- пересборка тестовой стратегии на phase review без явного `Spec Reopen`;
- редактирование spec-артефактов во время review;
- блокировка PR субъективными замечаниями без конкретного тестового риска.

## 6. Deliverables И Формат Результата

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-qa-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Требования к содержанию findings:
- `Issue`: конкретный test-gap, дефект тест-дизайна или риск нестабильности;
- `Impact`: риск пропуска регрессий, ложных результатов или нестабильности CI;
- `Suggested fix`: минимальный реалистичный способ исправления;
- `Spec reference`: ссылка на `70-test-plan.md` (и связанный spec-артефакт при необходимости).

## 7. Эскалация И Severity-Политика

`go-qa-review` использует стандартные severity (`critical/high/medium/low`) с QA-смыслом:
- `critical`:
  - отсутствуют тесты для критичных сценариев, обязательных по `70-test-plan.md`, что делает merge небезопасным;
  - тесты системно недетерминированы и ломают достоверность quality gates;
- `high`:
  - значимый пробел в покрытии по обязательным веткам;
  - ключевые assert-ы не проверяют требуемое поведение;
- `medium`:
  - неполнота edge/fail-path покрытия без немедленного merge-block, но с заметным риском;
- `low`:
  - локальные улучшения читабельности/поддерживаемости тестов.

Эскалация:
- если обнаружено расхождение между кодом и утвержденным test-plan/spec intent, оформляется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 8. Интерфейс Со Смежными Review Skills

- `go-idiomatic-review`: передает сигналы, где проблема касается стиля/структуры Go-кода за пределами QA-domain.
- `go-domain-invariant-review`: принимает эскалации, когда тесты не защищают критичные инварианты как бизнес-контракты.
- `go-reliability-review`: принимает эскалации по timeout/retry/degradation/shutdown поведению, если риск выходит за рамки test-quality.
- `go-performance-review`: принимает эскалации, если тестовый пробел связан с performance-регрессиями hot-path.
- `go-concurrency-review`: принимает эскалации по race/deadlock/goroutine lifecycle рискам.
- `go-db-cache-review`: принимает эскалации по data consistency/query/cache correctness.
- `go-security-review`: принимает эскалации по security negative-case coverage.
- `go-design-review`: финально валидирует, что test-layer изменения не нарушили design integrity.

Правило интерфейса:
- `go-qa-review` формулирует QA-риск и actionable fix, но не захватывает чужой primary-domain.

## 9. Definition Of Done Для Прохода Skill

Проход `go-qa-review` завершен, если:
- проверена полнота тестов относительно `70-test-plan.md` для измененного функционала;
- все `critical/high` QA-findings оформлены с `file:line`, impact, fix и spec reference;
- выявлены и явно отмечены риски flaky/недетерминированного поведения;
- нет неэскалированных spec-level конфликтов;
- review-вывод остается строго в QA-domain и не подменяет другие роли.

## 10. Анти-Паттерны

`go-qa-review` не должен:
- сводить проверку к формальному подсчету количества тестов без анализа качества проверок;
- дублировать глубокие выводы других reviewer-ролей вместо точечного handoff;
- предлагать переписывание архитектуры под видом тестового замечания;
- оставлять критичные test gaps без явной эскалации;
- принимать spec-level решения в фазе review без `Spec Reopen`.
