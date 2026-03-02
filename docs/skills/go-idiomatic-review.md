# Skill Spec: `go-idiomatic-review` (Domain-Scoped Review)

## 1. Назначение

`go-idiomatic-review` — экспертный review-skill по идиоматичности и инженерной дисциплине Go-кода в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса.

Ценность skill:
- находит дефекты и риски, которые проявляются как неидиоматичные Go-практики, а не как вкусовые замечания;
- удерживает код в предсказуемом стиле, совместимом с Go toolchain и модульными границами проекта;
- формирует actionable findings с конкретным `file:line`, impact и минимальным fix path.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за идиоматическое Go-review в рамках утвержденной спецификации.

## 2. Формат Hard Skills (как в AGENTS.md)

Для `go-idiomatic-review` hard skills задаются в том же формате, который используется в сильных skill-пакетах и в стиле `AGENTS.md`:
- `Mission`: что именно skill защищает на merge-path;
- `Default Posture`: какие инженерные предпосылки используются по умолчанию;
- доменные компетенции (`... Competency`) с проверяемыми правилами;
- `Review Blockers For This Skill`: что считается блокирующим для merge;
- явное разделение domain ownership и handoff в соседние review-skill.

Такой формат делает skill не только процессным, но и предметно-исполняемым.

## 3. Персонализированные Hard Skills Для `go-idiomatic-review`

### 3.1 Mission

- Защищать merge safety от Go-неидиоматичных решений, которые создают риск корректности, эксплуатации или сопровождения.
- Давать проверяемые idiomatic findings до глубоких доменных ревьюеров.
- Преобразовывать риск в минимальный конкретный path исправления без архитектурного redesign.

### 3.2 Default Posture

- Сначала review измененного diff и напрямую затронутых путей, а не «генеральная уборка» репозитория.
- Приоритет `correctness -> maintainability -> style consistency`, а не наоборот.
- Предпочтение явному control flow, явной обработке ошибок и явным границам модулей.
- Любые комментарии должны быть обоснованы через риск и Go-практику, а не через личный вкус.

### 3.3 Spec-First Review Competency

- Соблюдать ограничения Phase 4 из `docs/spec-first-workflow.md`:
  - domain-scoped review;
  - точные ссылки `file:line`;
  - практический fix path;
  - `Spec Reopen` при конфликте с утвержденным spec intent.
- Не редактировать spec-артефакты в Phase 4.
- Рассматривать незакрытые `critical/high` idiomatic findings как blockers для `Gate G4`.

### 3.4 Correctness-First Idiomatic Competency

- В первую очередь проверять поведенческие риски:
  - скрытое изменение API/контрактного поведения;
  - опасные nil/zero/default предпосылки;
  - рефакторы, которые усложняют reasoning о состоянии.
- Не подменять корректность «красивостью кода».

### 3.5 Control-Flow And Readability Competency

- Проверять ранние возвраты, минимальную вложенность, отсутствие лишних `else` после `return`.
- Фиксировать функции с несколькими уровнями абстракции и смешанными обязанностями.
- Отмечать неявный control flow (скрытые side effects, чрезмерная обертка), если это ухудшает предсказуемость.

### 3.6 Errors And Context Competency

- Ошибки: явные error-contract values, контекст операции, `%w` при необходимости unwrap, `errors.Is/As`, отказ от string-comparison.
- Контекст: `ctx` первым аргументом там, где есть cancel/deadline/scope; запрет хранения context в struct; обязательный `cancel()` для derived context.
- Запрет на подмену request-context на `context.Background()` в request flow.
- Явная обработка `context.Canceled` и `context.DeadlineExceeded` без маскировки под бизнес-ошибки.

### 3.7 Package/Module Boundary Competency

- Проверять дисциплину `cmd`/`internal`/`app`/`domain`/`infra` в соответствии с `docs/project-structure-and-module-organization.md`.
- Фиксировать мусорные пакеты (`util/common/helpers`) без предметной ответственности.
- Держать экспортируемую поверхность минимальной и осознанной.
- Отмечать скрытый DI через globals/init и нарушения import-direction.

### 3.8 Types/Interfaces/Pointer-Value Competency

- По умолчанию concrete types, interface только при доказанной потребности runtime-substitution.
- Малые consumer-owned interfaces вместо interface-per-struct.
- Осмысленные pointer/value решения, без pointer-to-basic и pointer-to-interface анти-паттернов.
- Проверка zero-value usability и отказ от преждевременных абстракций в стиле Java-overengineering.

### 3.9 Naming/Exports/Docs Competency

- Go naming conventions: package lowercase, non-stutter, initialisms (`ID`, `URL`, `HTTP`, `JSON`, `API`), короткие receiver names.
- Имена boolean как факты/вопросы (`isReady`, `hasNext`, `enabled`).
- При изменении exported API проверять наличие корректных doc comments и стабильность публичного контракта.

### 3.10 Toolchain And Validation Competency

- Рекомендовать проверку через репозиторные команды из `docs/build-test-and-development-commands.md`:
  - `make fmt-check`;
  - `make test`;
  - `go vet ./...`;
  - `make lint`;
  - `make test-race` при concurrency-surface.
- Не заявлять merge-readiness без явного validation path.

### 3.11 Trigger-Driven Cross-Domain Signal Competency

- Concurrency trigger: базовый idiomatic sanity check + handoff в `go-concurrency-review` для race/deadlock/leak.
- Test trigger: idiomatic test/readability-check + handoff в `go-qa-review` для полноты стратегии.
- Public API trigger: проверка naming/docs/export discipline + handoff в contract/design review для глубокой semantics-проверки.
- Performance trigger: evidence-first idiomatic check + handoff в `go-performance-review` для benchmark/profile глубины.
- Security trigger: фиксировать очевидные secure-coding anti-patterns + handoff в `go-security-review` для threat-depth анализа.

### 3.12 Evidence Threshold And Review Blockers

Каждая нетривиальная находка обязана содержать:
- `file:line`;
- нарушенное правило/ожидание;
- конкретный impact;
- минимальный путь исправления;
- как проверить исправление.

Merge-blockers для этого skill:
- потеря error/cause semantics или проглатывание ошибок;
- поломанная context-propagation/cancellation дисциплина;
- accidental export/package-boundary drift;
- control-flow сложность, скрывающая поведение;
- неидиоматичные абстракции без пользы;
- конфликт со spec intent без `Spec Reopen`.

## 4. Матрица Переноса Из Referenced Docs

| Источник | Что перенесено в hard skills | Где зафиксировано |
|---|---|---|
| `docs/spec-first-workflow.md` | Domain-scoped review, Gate G4 blockers, `Spec Reopen`, findings format | `Spec-First Review Competency`, `Evidence Threshold` |
| `docs/llm/go-instructions/70-go-review-checklist.md` | Review order, correctness-first posture, actionable output, validation baseline | `Default Posture`, `Correctness-First`, `Toolchain` |
| `docs/llm/go-instructions/10-go-errors-and-context.md` | `%w`, `errors.Is/As`, panic-policy, context-first signature, cancel discipline | `Errors And Context Competency` |
| `docs/llm/go-instructions/30-go-project-layout-and-modules.md` | package responsibility, `internal/`, export minimization, anti-`util/common` | `Package/Module Boundary Competency` |
| `docs/project-structure-and-module-organization.md` | repo-specific boundaries (`cmd`, `internal/app/domain/infra`) | `Package/Module Boundary Competency` |
| `docs/llm/go-instructions/20-go-concurrency.md` (trigger) | lifecycle/cancel/channel sanity signals | `Trigger-Driven Cross-Domain Signal` |
| `docs/llm/go-instructions/40-go-testing-and-quality.md` (trigger) | deterministic tests, race recommendation, quality checks | `Toolchain`, `Trigger-Driven Cross-Domain Signal` |
| `docs/build-test-and-development-commands.md` (trigger) | repo-native command set (`make fmt-check/test/test-race/lint`) | `Toolchain And Validation Competency` |
| `docs/llm/go-instructions/50-go-public-api-and-docs.md` (trigger) | export discipline, doc-comment standards, compatibility-first mindset | `Naming/Exports/Docs Competency` |
| `docs/llm/go-instructions/60-go-performance-and-profiling.md` (trigger) | evidence-first perf stance, no speculative complexity | `Trigger-Driven Cross-Domain Signal` |
| `docs/llm/security/10-secure-coding.md` (trigger) | obvious security anti-pattern signals in idiomatic review | `Trigger-Driven Cross-Domain Signal` |

## 5. Ответственность В Spec-First Workflow

Ключевая роль `go-idiomatic-review` — первый проход в Phase 4 review sequence, чтобы устранить high-impact idiomatic риски до более узких доменных проверок.

Обязательная ответственность в каждом проходе:
- выполнить domain-scoped review diff-а и затронутых файлов на идиоматичность и maintainability;
- фиксировать findings в workflow-формате (`severity`, `file:line`, `impact`, `suggested fix`, `spec reference`);
- отделять блокирующие проблемы (`critical/high`) от улучшений (`medium/low`);
- не редактировать spec-артефакты и не пересогласовывать архитектуру;
- при выявлении spec-mismatch инициировать `Spec Reopen`.

## 6. Границы Экспертизы (Out Of Scope)

`go-idiomatic-review` не подменяет соседние reviewer-роли:
- не валидирует бизнес-инварианты как primary-domain (`go-domain-invariant-review`);
- не проводит архитектурный redesign (`go-design-review`);
- не делает глубокий performance/concurrency/security/DB/reliability аудит как primary-domain;
- не владеет полнотой тестовой стратегии (`go-qa-review`).

Допустимо отмечать сигнал вне домена, но с явным handoff соответствующему review-skill.

## 7. Deliverables

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате:
  - `[severity] [go-idiomatic-review] [file:line]`
  - `Issue:`
  - `Impact:`
  - `Suggested fix:`
  - `Spec reference:`

Обязательные секции ответа skill:
- `Findings`
- `Handoffs`
- `Spec Reopen`
- `Residual Risks`
- `Validation commands`

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/project-structure-and-module-organization.md`

### 8.2 Trigger-Based

- Concurrency-surface: `docs/llm/go-instructions/20-go-concurrency.md`
- Test/quality-surface: `docs/llm/go-instructions/40-go-testing-and-quality.md`, `docs/build-test-and-development-commands.md`
- Export/public API changes: `docs/llm/go-instructions/50-go-public-api-and-docs.md`
- Performance-sensitive changes: `docs/llm/go-instructions/60-go-performance-and-profiling.md`
- Security-impacting idiomatic risks: `docs/llm/security/10-secure-coding.md`

## 9. Протокол Фиксации Findings

Каждую нетривиальную находку фиксировать в workflow-формате.
`IDM-###` можно использовать как внутренний трекинг-идентификатор (опционально):
1. `Где`: точный `file:line`.
2. `Что нарушено`: конкретное idiomatic правило/ожидание.
3. `Почему важно`: риск корректности/эксплуатации/поддерживаемости.
4. `Как исправить`: минимальный конкретный fix path.
5. `Как проверить`: релевантные команды/тесты.
6. `Нужна ли эскалация`: handoff или `Spec Reopen`.

## 10. Definition Of Done Для Прохода Skill

Проход `go-idiomatic-review` завершен, если:
- все `critical/high` idiomatic findings выявлены и actionably оформлены;
- нет вкусовых комментариев без risk-based аргументации;
- findings разделены на domain-owned и handoff;
- формат review-логов соответствует workflow;
- spec-конфликты отмечены через `Spec Reopen`, а не скрыты в комментариях.

## 11. Анти-Паттерны

`go-idiomatic-review` не должен:
- превращать review в субъективный style-policing;
- предлагать архитектурные переделки под видом «идиоматичности»;
- давать общие советы без `file:line` и конкретного fix path;
- смешивать домены без явного handoff;
- игнорировать контекст утвержденной спецификации и gate-правила.
