# Skill Spec: `go-db-cache-review` (Domain-Scoped Review)

## 1. Назначение

`go-db-cache-review` — экспертный review-skill по рискам и корректности DB/cache-путей в Phase 4 (`Domain-Scoped Code Review`) spec-first workflow.

Ценность skill:
- проверяет, что реализация DB/cache не нарушает утвержденный spec intent до merge;
- выявляет high-impact дефекты в query discipline, transaction boundaries и cache behavior;
- дает actionable findings в рамках своего домена без подмены соседних reviewer-ролей.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за DB/cache review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-db-cache-review` принимает решения по:
- SQL query discipline:
  - N+1/chatty-query риски;
  - неявные тяжелые query-path в hot/request path;
  - корректность параметризации, когда она влияет на безопасный data-access путь;
- transaction boundaries и consistency behavior:
  - корректность transactional scope;
  - отсутствие частичных side effects в связанных операциях;
  - соответствие transaction semantics утвержденному `40-data-consistency-cache.md`;
- context/timeouts/pooling hygiene для DB-вызовов:
  - корректная передача `context.Context`;
  - отсутствие бесконтрольных блокировок и pool starvation симптомов в коде;
- cache correctness:
  - ключи, tenant/scope isolation, versioning/namespace hygiene;
  - TTL/staleness behavior против spec-контракта;
  - invalidation/update paths при write-операциях;
- cache stampede/degradation behavior:
  - риски thundering herd и отсутствие защитных паттернов в критичных path;
  - корректность fallback/bypass логики в пределах утвержденного spec intent.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-db-cache-review`:
- выполнять Phase 4 проверку при наличии DB/cache изменений;
- валидировать соответствие реализации артефактам `40/60/70/90` и затронутым частям `30/50/55`.

Обязательная ответственность в каждом проходе:
- оставлять findings только в DB/cache-domain;
- ссылаться на конкретный `file:line` и `Spec reference`;
- давать practical fix path, а не общий совет;
- не редактировать spec-файлы в review-фазе;
- оформлять `Spec Reopen`, если дефект невозможно исправить без изменения утвержденного spec intent.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Query Discipline`:
  - отсутствуют ли новые N+1/query-in-loop паттерны;
  - не добавлен ли лишний DB round-trip в критичные path без явного обоснования;
- `Transaction Boundary Correctness`:
  - корректно ли сгруппированы read-write операции в транзакции;
  - нет ли рисков partial commit/partial side effect;
- `DB Context And Resource Safety`:
  - есть ли cancel/deadline-aware DB calls;
  - корректно ли освобождаются ресурсы (`rows/tx` lifecycle);
- `Cache Key And Isolation Correctness`:
  - ключи учитывают tenant/scope/version, когда это требуется контрактом;
  - нет ли риска cross-tenant/cross-scope data mix;
- `Invalidation And Staleness Contract`:
  - write-path корректно синхронизирован с invalidation/update cache entries;
  - staleness behavior соответствует зафиксированному API/domain ожиданию;
- `Stampede And Degradation Controls`:
  - нет ли очевидного stampede-risk в miss-path;
  - fallback/bypass поведение не противоречит утвержденной reliability-политике;
- `DB/Cache Test Readiness Signals`:
  - для критичных DB/cache веток есть проверяемые тестовые сигналы (в объеме, заданном `70-test-plan.md`).

## 5. Границы Экспертизы (Out Of Scope)

`go-db-cache-review` не подменяет соседние review-роли:
- не делает полный idiomatic/language-style review (`go-idiomatic-review`, `go-language-simplifier-review`);
- не делает primary performance evidence review (`go-performance-review`), кроме прямых DB/cache regression-сигналов;
- не делает primary concurrency correctness review (`go-concurrency-review`), кроме точек, где concurrency влияет на cache correctness;
- не делает primary reliability policy review (`go-reliability-review`);
- не делает primary security review (`go-security-review`);
- не делает общий architecture integrity review (`go-design-review`);
- не делает тест-стратегический аудит как primary-domain (`go-qa-review`).

Также вне scope:
- redesign утвержденной архитектуры без явного spec-конфликта;
- редактирование specification-артефактов в review-фазе;
- замечания без конкретного DB/cache риска и без привязки к коду.

## 6. Интерфейс Со Смежными Review Skills

`go-db-cache-review` передает handoff:
- в `go-performance-review`, если root issue требует benchmark/profile-level доказательств latency/throughput;
- в `go-concurrency-review`, если основной риск в goroutine/channel/lock механике, а DB/cache является симптомом;
- в `go-reliability-review`, если дефект в timeout/retry/degradation/shutdown policy;
- в `go-security-review`, если проблема касается authz/tenant isolation/secret leakage/injection класса;
- в `go-qa-review`, если основной gap в тестовом покрытии DB/cache-сценариев;
- в `go-design-review`, если исправление требует архитектурного изменения вне текущего spec intent.

Правило интерфейса:
- `go-db-cache-review` фиксирует DB/cache root risk и минимально достаточный safe fix,
- но не захватывает primary-domain соседнего review-skill.

## 7. Deliverable Формат Для Review-Лога

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-db-cache-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Минимальные требования к finding:
- `Issue`: конкретный DB/cache дефект или риск;
- `Impact`: как это влияет на корректность данных, согласованность, отказоустойчивость или merge-safety;
- `Suggested fix`: реалистичный минимальный путь исправления;
- `Spec reference`: ссылка на релевантный spec-артефакт (обычно `40/55/70/90`, при необходимости `30/60/50`).

## 8. Severity И Эскалация

Severity-интерпретация в DB/cache-domain:
- `critical`:
  - подтвержденный риск некорректности данных/несогласованности/опасного stale behavior в критичном path;
  - дефект, блокирующий безопасный merge до исправления;
- `high`:
  - высокая вероятность N+1/chatty-query деградации в значимом path;
  - высокая вероятность invalidation/consistency ошибки при типичных операциях;
- `medium`:
  - локальный DB/cache риск с ограниченным blast radius;
- `low`:
  - улучшения качества и предсказуемости без немедленного merge-block.

Эскалация:
- если safe fix требует изменить утвержденный spec intent, инициируется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 9. Definition Of Done Для Прохода Skill

Проход `go-db-cache-review` завершен, если:
- проверены все измененные DB/cache участки по обязательному scope;
- все `critical/high` findings оформлены с `file:line`, impact, suggested fix и spec reference;
- кросс-доменные риски переданы через handoff в профильные reviewer-роли;
- нет неэскалированных spec-level конфликтов;
- при отсутствии проблем явно зафиксировано, что DB/cache findings не обнаружены.

## 10. Анти-Паттерны

`go-db-cache-review` не должен:
- превращаться в общий code review без DB/cache фокуса;
- блокировать merge по субъективным предпочтениям без явного data/cache риска;
- предлагать redesign вместо явного `Spec Reopen`;
- игнорировать cache invalidation/staleness контракты и смотреть только на happy-path;
- оставлять подтвержденный DB consistency риск без фиксации или эскалации.

## 11. Статус Текущего Документа

Этот файл фиксирует `SCOPE` и `RESPONSIBILITIES` для будущего `SKILL.md` по `go-db-cache-review`.
Runtime-инструкция (`Working Rules`, `Context Intake`, финальный execution protocol) оформляется отдельным шагом.
