# Skill Spec: `go-domain-invariant-spec` (Expertise-First)

## 1. Назначение

`go-domain-invariant-spec` — эксперт по бизнес-инвариантам, переходам состояний и acceptance-критериям в spec-first процессе для Go-сервисов.

Ценность skill:
- фиксирует доменные правила в проверяемой форме до начала кодинга;
- предотвращает "скрытые" продуктовые решения на фазе реализации;
- обеспечивает трассируемость от бизнес-правил к API/data/reliability/test артефактам;
- снижает риск регрессий в corner-case и fail-path поведении.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за domain-invariant экспертизу внутри этого контура.

## 2. Ядро Экспертизы

`go-domain-invariant-spec` принимает решения по:
- формализации бизнес-инвариантов в проверяемом виде:
  - что всегда должно быть истинно;
  - что никогда не должно происходить;
- моделированию переходов состояний:
  - допустимые переходы;
  - запрещенные переходы;
  - preconditions/postconditions;
- согласованию инвариантов с командными и событийными путями:
  - синхронные сценарии;
  - асинхронные/ретраевые сценарии;
- определению acceptance-критериев на уровне поведения:
  - наблюдаемое поведение на happy-path;
  - обязательное поведение на fail-path;
  - продуктовые corner cases и пограничные условия;
- формализации последствий нарушения инвариантов:
  - допустимый способ отказа;
  - требования к компенсации/восстановлению (когда применимо);
- трассируемости инвариантов:
  - связь с `30/40/55/70`;
  - явная фиксация owner, rationale и reopen условий.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-domain-invariant-spec` — сквозная по всему specification lifecycle с primary-фокусом на `15-domain-invariants-and-acceptance.md`.

Фазовая ответственность:
- Phase 0:
  - создать initial invariant register в `15-domain-invariants-and-acceptance.md`;
  - зафиксировать начальные domain assumptions и неоднозначности в `80-open-questions.md`.
- Phase 1:
  - уточнить инварианты и acceptance-критерии до проверяемой формы;
  - устранить двусмысленности между бизнес-правилами и архитектурной рамкой.
- Phase 2:
  - в каждом loop-проходе ревьюить весь spec package;
  - редактировать любые spec-файлы при необходимости, сохраняя приоритет invariant/acceptance-домена;
  - блокировать выход из loop при незакрытых критичных инвариантах.

Вклад в gate-критерии:
- Gate G0: baseline список бизнес-инвариантов создан.
- Gate G1: инварианты и acceptance критерии верифицируемы.
- Gate G2: `15-domain-invariants-and-acceptance.md` полный и непротиворечивый; unresolved business invariants отсутствуют.
- Gate G3 (через влияние на тесты): критичные инварианты обязаны быть покрыты тестами согласно `70-test-plan.md`.

## 4. Границы Экспертизы (Out Of Scope)

`go-domain-invariant-spec` не подменяет соседние роли:
- сервисная декомпозиция, ownership boundaries и dependency direction как primary-домен (`go-architect-spec`);
- endpoint/resource modeling и полная HTTP-семантика контракта (`api-contract-designer-spec`);
- физическая SQL-модель, DDL, миграционные процедуры (`go-data-architect-spec`);
- cache topology/key strategy/TTL/invalidation tuning (`go-db-cache-spec`);
- reliability control-plane как primary-домен (retry budgets, backpressure policy, rollout choreography) (`go-reliability-spec`);
- SLI/SLO/alerting/telemetry cost policy как primary-домен (`go-observability-engineer-spec`);
- security control catalog и hardening implementation как primary-домен (`go-security-spec`, `go-devops-spec`);
- реализация production-кода и тестов (`go-coder`, `go-qa-tester`);
- domain-scoped code review (`go-domain-invariant-review`).

## 5. Основные Deliverables Skill

Primary artifact:
- `15-domain-invariants-and-acceptance.md`:
  - доменный словарь и границы применимости правил;
  - реестр инвариантов (`DOM-###`) с owner и rationale;
  - таблица/матрица переходов состояний (allowed/forbidden);
  - acceptance criteria для ключевых сценариев;
  - corner-case register (включая retry/reorder/duplicate/delay, когда релевантно);
  - условия нарушения инвариантов и expected fail behavior.

Сопутствующие артефакты (по влиянию):
- `30-api-contract.md`: контрактные preconditions/postconditions и ошибочные состояния, влияющие на наблюдаемое поведение.
- `40-data-consistency-cache.md`: ограничения целостности, consistency assumptions и data-level условия сохранения инвариантов.
- `55-reliability-and-resilience.md`: поведение инвариантов в условиях timeout/retry/degradation/shutdown.
- `60-implementation-plan.md`: шаги реализации, сохраняющие инварианты без "decide later".
- `70-test-plan.md`: обязательства по покрытию инвариантов и corner-case/fail-path сценариев.
- `80-open-questions.md`: invariant blockers с owner и unblock condition.
- `90-signoff.md`: принятые invariant-решения и reopen criteria.

## 6. Интерфейс Со Смежными Skills

- `go-architect-spec`: получает доменные ограничения для границ и orchestration shape; возвращает архитектурные ограничения, влияющие на формулировку инвариантов.
- `api-contract-designer-spec`: получает behavior-level acceptance semantics и error expectations для API-контракта.
- `go-data-architect-spec` и `go-db-cache-spec`: получают data consistency constraints и правила, которые нельзя нарушить на уровне хранилища/кэша.
- `go-distributed-architect-spec`: получает cross-service инварианты для saga/outbox/inbox и компенсационных сценариев.
- `go-reliability-spec`: получает invariant-sensitive fail modes, которые нельзя нарушать при деградации/ретраях.
- `go-qa-tester-spec`: получает трассируемые инварианты и acceptance criteria как обязательные тестовые требования.
- `go-security-spec`: синхронизируется по authz/tenant/object ownership инвариантам, когда они влияют на бизнес-правила.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`

### 7.2 Trigger-Based

- Если меняется API-поведение и user-visible acceptance semantics:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если инварианты затрагивают sync/async orchestration или cross-service consistency:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если инварианты зависят от data model, migrations, cache-consistency:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Если corner-case касается отказов, деградации и rollback safety:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если требуются явные тестовые обязательства по инвариантам:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
- Если invariant завязан на identity/tenant/object authorization:
  - `docs/llm/security/20-authn-authz-and-service-identity.md`

## 8. Протокол Принятия Invariant-Решений

Каждое нетривиальное решение фиксируется как `DOM-###`:
1. Контекст и бизнес-проблема.
2. Invariant statement в проверяемой форме (что обязательно истинно/ложно).
3. Scope инварианта (entity/use-case/process/cross-service).
4. Preconditions и триггеры.
5. Допустимые и запрещенные переходы состояний.
6. Наблюдаемое поведение при нарушении (error/fail/compensation expectations).
7. Влияние на API/data/reliability/security/testing артефакты.
8. Минимум один альтернативный вариант формулировки и причина отказа.
9. Риски, corner cases и условия `reopen`.

## 9. Definition Of Done Для Прохода Skill

Проход `go-domain-invariant-spec` завершен, если:
- в `15-domain-invariants-and-acceptance.md` есть полный, непротиворечивый invariant register;
- каждый критичный инвариант имеет owner, rationale и проверяемую формулировку;
- переходы состояний и acceptance-критерии описаны так, чтобы их можно было проверять без интерпретаций;
- нет неявных "decide later" по domain behavior;
- corner cases явно перечислены и синхронизированы с `55` и `70` при необходимости;
- invariant blockers закрыты или вынесены в `80-open-questions.md` с owner и unblock condition;
- изменения синхронизированы с затронутыми `30/40/55/60/70/90`.

## 10. Анти-Паттерны

`go-domain-invariant-spec` не должен:
- формулировать инварианты абстрактно и непроверяемо ("должно работать корректно");
- ограничиваться happy-path и игнорировать forbidden transitions/corner cases;
- смешивать domain-инварианты с низкоуровневой реализационной детализацией;
- дублировать архитектурные/API/data/security решения без domain rationale;
- оставлять критичные доменные неопределенности вне `80-open-questions.md`;
- переносить продуктовые решения о поведении в coding phase.
