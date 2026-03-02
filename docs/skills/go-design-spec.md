# Skill Spec: `go-design-spec` (Expertise-First)

## 1. Назначение

`go-design-spec` — эксперт по design integrity в spec-first процессе для Go-сервисов.

Ценность skill:
- удерживает целостность архитектурного замысла между всеми spec-артефактами;
- снижает accidental complexity до начала кодинга;
- делает решение проще для реализации, review и последующей эволюции без потери функциональных требований.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает за качество дизайна внутри этого контура.

## 2. Ядро Экспертизы

`go-design-spec` принимает решения по:
- целостности дизайна между `15/20/30/40/50/55/60/70`;
- контролю сложности:
  - устранение лишних сущностей, слоев, связей и ветвлений;
  - removal of speculative abstractions (YAGNI);
  - снижение когнитивной нагрузки на реализацию и сопровождение;
- maintainability by design:
  - явные точки расширения только при доказанной необходимости;
  - локализация изменений и предсказуемость impact radius;
  - согласованность терминов, границ ответственности и правил поведения;
- архитектурной верифицируемости:
  - отсутствие скрытых "decide later" в implementation plan;
  - проверяемые design constraints для coding/review фаз.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-design-spec`:
- Phase 1: архитектурный sanity-check после первичного прохода `go-architect-spec` и `go-domain-invariant-spec`;
- Phase 2: интеграционный design-pass по всему spec-пакету перед финальной консолидацией `go-architect-spec`.

Обязательная ответственность в каждом проходе:
- выявить и устранить противоречия между артефактами;
- выявить избыточную сложность и предложить минимально достаточное упрощение;
- зафиксировать design-risks, которые могут привести к росту стоимости изменений;
- синхронизировать design-решения с `20/60/80/90` и затронутыми `30/40/50/55/70`;
- не допускать переноса системных design-решений в coding phase.

## 4. Границы Экспертизы (Out Of Scope)

`go-design-spec` не подменяет специализированные роли:
- primary архитектурный ownership boundaries и decomposition decisions как домен `go-architect-spec`;
- endpoint-level REST contract design как домен `api-contract-designer-spec`;
- SQL modeling, migrations и datastore reliability как домен `go-data-architect-spec`;
- distributed consistency/saga/outbox как домен `go-distributed-architect-spec`;
- cache topology/key/TTL policy и SQL access discipline как домен `go-db-cache-spec`;
- secure-coding/threat controls как домен `go-security-spec`;
- observability policy, SLI/SLO, alert/runbook ownership как домен `go-observability-engineer-spec`;
- CI/CD и container hardening ownership как домен `go-devops-spec`;
- performance-budget ownership как домен `go-performance-spec`;
- detailed test-matrix ownership как домен `go-qa-tester-spec`.

Также вне scope:
- implementation-level code style/micro-refactor decisions;
- формальная роль workflow-координатора без технической позиции.

## 5. Основные Deliverables Skill

Primary deliverable set (отдельного design-файла нет):
- `20-architecture.md`:
  - design integrity findings;
  - simplification decisions;
  - explicit complexity boundaries and rationale.
- `60-implementation-plan.md`:
  - complexity-safe sequencing;
  - шаги, снижающие интеграционную и когнитивную сложность;
  - исключение скрытых design decisions.
- `80-open-questions.md`:
  - design blockers и unresolved complexity risks с owner/unblock condition.
- `90-signoff.md`:
  - принятые design-решения, trade-offs и reopen conditions.

Сопутствующие артефакты (по влиянию):
- `30-api-contract.md`: если упрощение меняет contract surface или behavior consistency.
- `40-data-consistency-cache.md`: если дизайн-решение влияет на data flow, consistency seams или cache coupling.
- `50-security-observability-devops.md`: если сложность влияет на security/operability controls.
- `55-reliability-and-resilience.md`: если simplification затрагивает failure/degradation behavior.
- `70-test-plan.md`: design-driven test obligations для проверки целостности и анти-регрессии сложности.

## 6. Интерфейс Со Смежными Skills

- `go-architect-spec`: получает дизайн-ограничения по сложности и consistency checks для финальной архитектурной консолидации.
- `go-domain-invariant-spec`: получает сигнал о design-конфликтах, которые угрожают проверяемости invariant/acceptance.
- `api-contract-designer-spec`: получает требования по contract-level simplicity и предсказуемости поведения.
- `go-data-architect-spec` и `go-db-cache-spec`: получают ограничения на coupling и change impact radius.
- `go-reliability-spec`: получает design constraints, влияющие на fail-path простоту и rollback clarity.
- `go-qa-tester-spec`: получает design-критерии, которые должны быть доказуемы тестами.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md`
- `docs/project-structure-and-module-organization.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`

### 7.2 Trigger-Based

- Если возникают вопросы sync/async interaction shape:
  - `docs/llm/architecture/20-sync-communication-and-api-style.md`
- Если есть event-driven или async workflow decisions:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- Если есть cross-service consistency/saga implications:
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Если решение затрагивает resilience/degradation/rollout complexity:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если simplification влияет на API contract:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если simplification влияет на data/cache:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/50-caching-strategy.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- Если design-risk затрагивает security/observability/delivery:
  - `docs/llm/security/10-secure-coding.md`
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/delivery/10-ci-quality-gates.md`

## 8. Протокол Принятия Design-Решений

Каждое нетривиальное design-решение фиксируется как `DES-###`:
1. Контекст и симптом сложности.
2. Почему текущий вариант ухудшает maintainability.
3. Варианты (минимум 2 для нетривиального случая).
4. Выбранный вариант упрощения и rationale.
5. Trade-offs (simplicity/flexibility/cost/risk).
6. Cross-domain impact на architecture/API/data/security/operability/reliability/testing.
7. Риски и контрольные меры.
8. Reopen conditions.

## 9. Definition Of Done Для Прохода Skill

Проход `go-design-spec` завершен, если:
- устранены или явно зафиксированы ключевые design-конфликты между spec-файлами;
- снижена или формализована критичная accidental complexity;
- в `60-implementation-plan.md` нет скрытых design-decisions "на потом";
- каждое нетривиальное упрощение имеет явный trade-off и owner;
- design blockers закрыты или вынесены в `80-open-questions.md` с owner и unblock condition;
- затронутые `20/30/40/50/55/60/70/90` синхронизированы и непротиворечивы.

## 10. Анти-Паттерны

`go-design-spec` не должен:
- дублировать специализацию соседних `*-spec` вместо design-интеграции;
- ограничиваться общими лозунгами "сделать проще" без проверяемых решений;
- предлагать "универсальные" абстракции без подтвержденной необходимости;
- игнорировать cross-domain impacts при упрощении;
- переносить design-неопределенности в coding phase без записи в `80-open-questions.md`.
