# Skill Spec: `go-domain-invariant-review` (Domain-Scoped Review)

## 1. Назначение

`go-domain-invariant-review` — экспертный review-skill по проверке сохранения бизнес-инвариантов, корректности переходов состояний и соблюдения acceptance-поведения в Phase 4 spec-first процесса для Go-сервисов.

Ценность skill:
- выявляет расхождения между реализацией и утвержденным доменным поведением из spec-артефактов;
- предотвращает merge изменений, которые нарушают критичные инварианты на happy-path, fail-path и corner-case сценариях;
- обеспечивает трассируемую, actionable обратную связь в формате review workflow.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за domain-invariant review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-domain-invariant-review` принимает решения по:
- фактическому сохранению инвариантов из `15-domain-invariants-and-acceptance.md` в коде;
- корректности state-transition логики:
  - допустимые переходы действительно разрешены;
  - запрещенные переходы действительно блокируются;
  - preconditions/postconditions соблюдаются;
- соблюдению acceptance-критериев в наблюдаемом поведении:
  - happy-path;
  - fail-path;
  - corner cases;
- корректности invariant-violation semantics:
  - предсказуемый отказ и error behavior;
  - отсутствие неуправляемых partial side effects;
- трассируемости от кода и тестов к `DOM-###` решениям и acceptance-критериям;
- выявлению spec-level рассогласований, требующих `Spec Reopen`.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-domain-invariant-review`:
- Phase 4 (`Domain-Scoped Code Review`) с primary-фокусом на соответствие реализации артефактам `15/30/40/55/70/90`;
- подтверждение, что реализация не нарушает утвержденные бизнес-инварианты после завершения coding-фазы.

Обязательная ответственность в каждом проходе:
- оставлять findings только в domain-invariant домене;
- ссылаться на конкретный `file:line` и `Spec reference`;
- давать практический fix path, а не общие рекомендации;
- не редактировать spec-файлы во время code review;
- при spec-конфликте создавать `Spec Reopen` запись в `reviews/<feature-id>/code-review-log.md`.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Invariant Preservation`:
  - реализованы ли проверки, которые гарантируют утвержденные `DOM-###`;
  - отсутствуют ли код-пути, обходящие invariant guards;
- `State Transition Correctness`:
  - не допускает ли код forbidden transitions;
  - корректно ли выражены preconditions/postconditions в use-case orchestration;
- `Acceptance Behavior Conformance`:
  - соответствует ли наблюдаемое API/service behavior acceptance-критериям;
  - корректно ли реализованы domain-level ошибки при нарушении инвариантов;
- `Corner-Case and Fail-Path Coverage`:
  - учитываются ли важные retry/reorder/duplicate/delay/failure сценарии, когда они определены в spec;
  - есть ли риски silent corruption или неявной потери доменной целостности;
- `Test Traceability`:
  - покрыты ли критичные инварианты и переходы тестами в соответствии с `70-test-plan.md`;
  - нет ли расхождения между заявленной и фактической проверяемостью инвариантов;
- `Spec Consistency`:
  - не появились ли в коде неутвержденные продуктовые решения о доменном поведении;
  - если появились, зафиксирован ли `Spec Reopen`.

## 5. Границы Экспертизы (Out Of Scope)

`go-domain-invariant-review` не подменяет смежные review-роли:
- idiomatic/style/language simplification как primary-domain (`go-idiomatic-review`, `go-language-simplifier-review`);
- архитектурная целостность и complexity control как primary-domain (`go-design-review`);
- performance profiling/hot-path regressions как primary-domain (`go-performance-review`);
- goroutine/race/deadlock/lifecycle аудит как primary-domain (`go-concurrency-review`);
- query discipline, transaction internals, cache invalidation correctness как primary-domain (`go-db-cache-review`);
- timeout/retry/backpressure/degradation/shutdown/rollback correctness как primary-domain (`go-reliability-review`);
- secure coding и threat controls как primary-domain (`go-security-review`);
- полнота тестовой стратегии как primary-domain (`go-qa-review`).

Допустимо подать сигнал в смежный домен, если invariant-риск имеет кросс-доменную причину, но ownership глубокого анализа остается у профильного review-skill.

## 6. Интерфейс Со Смежными Review Skills

`go-domain-invariant-review`:
- принимает как вход:
  - утвержденный invariant register и acceptance-критерии (`15`);
  - контракты/API-семантику (`30`), data/consistency ограничения (`40`), reliability-поведение (`55`), test obligations (`70`), sign-off решения (`90`);
- передает handoff-сигналы в:
  - `go-db-cache-review`, если риск нарушения инварианта связан с transaction/query/cache path;
  - `go-reliability-review`, если риск проявляется через retry/timeout/degradation behavior;
  - `go-security-review`, если invariant зависит от authz/tenant/object ownership enforcement;
  - `go-qa-review`, если проблема в недостающем/некачественном покрытии;
  - `go-design-review`, если нарушение вызвано архитектурным drift.

Правило интерфейса:
- skill формулирует invariant-impact и требуемый handoff, но не захватывает чужой primary-domain глубже необходимого для обоснования риска.

## 7. Deliverables И Формат Результата

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-domain-invariant-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Требования к каждому finding:
- `Issue`: какое invariant/transition/acceptance правило нарушено;
- `Impact`: доменное последствие (некорректное состояние, нарушение бизнес-ограничений, неконсистентное поведение);
- `Suggested fix`: минимальный безопасный способ исправления;
- `Spec reference`: ссылка минимум на один релевантный источник из `15/30/40/55/70/90`.

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md` (Phase 4, Reviewer Focus Matrix, Findings Format, Gate G4)
- `specs/<feature-id>/15-domain-invariants-and-acceptance.md`
- `specs/<feature-id>/70-test-plan.md`
- `specs/<feature-id>/90-signoff.md`

### 8.2 Trigger-Based

- Если изменение влияет на API-visible acceptance semantics:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если инвариант зависит от data consistency/cache behavior:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Если задействованы async/saga/reconciliation сценарии:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если риск проявляется через retry/timeout/degradation behavior:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если инвариант зависит от authz/tenant boundaries:
  - `docs/llm/security/20-authn-authz-and-service-identity.md`
- Если требуется сверка качества invariant-тестов:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`

## 9. Эскалация И Severity-Политика

`go-domain-invariant-review` использует стандартные severity (`critical/high/medium/low`) в доменном смысле:
- `critical`:
  - реализация допускает нарушение критичного инварианта;
  - допускается запрещенный переход состояния;
  - поведение противоречит утвержденным acceptance-критериям и создает merge-blocker;
- `high`:
  - высокая вероятность нарушения инварианта в fail-path/corner-case;
  - существенное расхождение с `15/70`, требующее исправления до merge;
- `medium`:
  - заметный доменный риск с ограниченным радиусом, не являющийся немедленным блокером;
- `low`:
  - локальное улучшение формулировки или трассируемости без прямого блокирующего эффекта.

Эскалация:
- при конфликте с утвержденной спецификацией оформляется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 10. Definition Of Done Для Прохода Skill

Проход `go-domain-invariant-review` завершен, если:
- выполнена проверка сохранения критичных инвариантов и переходов состояний против `15`;
- findings оформлены в workflow-формате с `file:line`, `Impact`, `Suggested fix`, `Spec reference`;
- все блокирующие (`critical/high`) domain findings либо исправлены, либо эскалированы через `Spec Reopen`;
- нет неэскалированных spec-level противоречий;
- review-вывод не выходит за границы domain-invariant домена.

## 11. Анти-Паттерны

`go-domain-invariant-review` не должен:
- превращаться в общий "find anything" review без domain-фокуса;
- подменять доменную проверку style/architecture/performance ревью-комментариями;
- давать абстрактные замечания без привязки к invariant-правилу и `file:line`;
- игнорировать corner-case/fail-path сценарии, ограничиваясь только happy-path;
- оставлять spec-конфликт без явной эскалации через `Spec Reopen`.
