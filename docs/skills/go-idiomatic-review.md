# Skill Spec: `go-idiomatic-review` (Domain-Scoped Review)

## 1. Назначение

`go-idiomatic-review` — экспертный review-skill по идиоматичности и инженерной дисциплине Go-кода в Phase 4 (`Domain-Scoped Code Review`) spec-first процесса.

Ценность skill:
- находит дефекты и риски, которые проявляются как неидиоматичные Go-практики (а не как вкусовые предпочтения);
- удерживает код в предсказуемом, поддерживаемом стиле, совместимом с Go toolchain;
- дает actionable findings с конкретным путём исправления и проверкой.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за идиоматическое Go-review в рамках утвержденной спецификации.

## 2. Ядро Экспертизы

`go-idiomatic-review` отвечает за проверку и оценку:
- idiomatic control flow:
  - ранние возвраты, минимальная вложенность, отсутствие лишних `else` после `return`;
  - читаемость happy-path и локальность обработки ошибок;
- errors и context:
  - корректная передача/оборачивание ошибок (`%w`, `errors.Is/As`);
  - отсутствие string-based сравнения ошибок;
  - корректная propagation `context.Context`, таймаутов и отмены;
- package/API discipline:
  - чистые package boundaries, отсутствие «мусорных» `util/common` пакетов;
  - минимальная и осознанная экспортируемая поверхность;
  - соответствие структуры проектным модульным правилам;
- idiomatic types and interfaces:
  - осмысленное применение интерфейсов, pointer/value семантики, zero-value usability;
  - отсутствие преждевременных абстракций и Java-style overengineering;
- naming/readability/toolchain hygiene:
  - Go naming conventions (initialisms, receiver names, package naming);
  - совместимость с `gofmt/goimports` и нормальной практикой Go-команд.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-idiomatic-review` — первый проход в Phase 4 review sequence, чтобы устранить high-impact idiomatic риски до более узких доменных проверок.

Обязательная ответственность в каждом проходе:
- выполнить domain-scoped review diff-а и затронутых файлов на идиоматичность и maintainability;
- фиксировать findings в формате workflow (`severity`, `file:line`, `impact`, `suggested fix`, `spec reference`);
- отделять блокирующие проблемы (`critical/high`) от улучшений (`medium/low`);
- не редактировать spec-артефакты и не пересогласовывать архитектуру;
- при выявлении spec-mismatch инициировать `Spec Reopen` запись, а не «чинить спецификацию в ревью».

## 4. Границы Экспертизы (Out Of Scope)

`go-idiomatic-review` не подменяет соседние reviewer-роли:
- не валидирует бизнес-инварианты как primary-домен (`go-domain-invariant-review`);
- не проводит архитектурный redesign (`go-design-review`);
- не выполняет глубокий performance-аудит с budget/benchmark ownership (`go-performance-review`);
- не выполняет специализированный concurrency-аудит (`go-concurrency-review`);
- не выполняет security-аудит как primary-домен (`go-security-review`);
- не владеет полнотой тестовой стратегии/traceability к `70-test-plan.md` (`go-qa-review`);
- не владеет DB/cache correctness и transaction discipline как primary-домен (`go-db-cache-review`);
- не владеет reliability-политиками timeout/retry/degradation как primary-домен (`go-reliability-review`).

Допустимо отметить риск вне домена, но без ухода в глубокую экспертизу: issue помечается как handoff соответствующему review-skill.

## 5. Основные Deliverables Skill

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` по workflow-формату:
  - `[severity] [go-idiomatic-review] [file:line]`
  - `Issue:`
  - `Impact:`
  - `Suggested fix:`
  - `Spec reference:`

Качество deliverable:
- каждая блокирующая находка должна иметь проверяемое объяснение "почему это неидиоматично/рискованно";
- каждая рекомендация должна быть реализуема без архитектурной переизобретательности;
- findings не должны противоречить утвержденному spec без явной `Spec Reopen` эскалации.

## 6. Интерфейс Со Смежными Review Skills

- `go-design-review`: принимает эскалации, где idiomatic issue связан с архитектурным drift.
- `go-domain-invariant-review`: принимает эскалации, где стиль скрывает или ломает доменную корректность.
- `go-performance-review`: принимает эскалации по hot-path/perf regressions, если проблема выходит за рамки idiomatic guidance.
- `go-concurrency-review`: принимает эскалации по lifecycle/race/deadlock/leak рискам.
- `go-db-cache-review`: принимает эскалации по query/transaction/cache correctness.
- `go-reliability-review`: принимает эскалации по timeout/retry/shutdown/degradation поведению.
- `go-security-review`: принимает эскалации по security-class issues.
- `go-qa-review`: принимает эскалации по coverage/traceability/missing tests.

Правило интерфейса: `go-idiomatic-review` формулирует сигнал и impact, но не захватывает чужой primary-domain.

## 7. Матрица Документов Для Экспертизы

### 7.1 Always

- `docs/spec-first-workflow.md` (разделы reviewer scope, findings format, readiness)
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/project-structure-and-module-organization.md`

### 7.2 Trigger-Based

- Если затронуты goroutines/channels/mutex/worker lifecycle:
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Если затронуты тесты или нужен quality-gate контекст:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Если меняется экспортируемая API surface:
  - `docs/llm/go-instructions/50-go-public-api-and-docs.md`
- Если заявлены performance-sensitive изменения:
  - `docs/llm/go-instructions/60-go-performance-and-profiling.md`
- Если идиоматический риск пересекается с secure coding:
  - `docs/llm/security/10-secure-coding.md`

## 8. Протокол Фиксации Findings

Каждую нетривиальную находку фиксировать как `IDM-###` с минимальным набором:
1. `Где`: точный `file:line`.
2. `Что нарушено`: конкретное idiomatic правило/практика.
3. `Почему важно`: риск корректности/поддерживаемости/операционной предсказуемости.
4. `Как исправить`: минимальный и конкретный путь исправления.
5. `Как проверить`: команды/тесты/линтер-проверки (по релевантности).
6. `Нужна ли эскалация`: handoff в другой review-domain при выходе за границы идиоматики.

## 9. Definition Of Done Для Прохода Skill

Проход `go-idiomatic-review` завершен, если:
- все `critical/high` idiomatic findings выявлены и формализованы actionably;
- нет неаргументированных «вкусовых» замечаний без impact;
- findings разделены на domain-owned и handoff-эскалации;
- формат review-логов соответствует workflow;
- при spec-конфликте создан `Spec Reopen` вместо неявного изменения требований.

## 10. Анти-Паттерны

`go-idiomatic-review` не должен:
- превращать review в субъективный style-policing без связи с риском;
- предлагать архитектурные переделки под видом «идиоматичности»;
- давать общие советы без точек кода и без конкретного fix path;
- смешивать ошибки из других доменов без явного handoff;
- игнорировать контекст утвержденной спецификации и gate-правил;
- рекомендовать изменения, несовместимые с Go toolchain и правилами проекта.
