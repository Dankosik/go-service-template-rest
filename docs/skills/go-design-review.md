# Skill Spec: `go-design-review` (Expertise-First)

## 1. Назначение

`go-design-review` — эксперт по design integrity на фазе code review в spec-first процессе для Go-сервисов.

Ценность skill:
- проверяет соответствие реализации утвержденному архитектурному и design-замыслу;
- предотвращает architectural drift после `Gate G2` и во время `Spec Freeze`;
- удерживает maintainability и контролирует accidental complexity в коде до merge.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за design-review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-design-review` принимает решения по:
- соответствию кода утвержденным решениям из `20-architecture.md` и `60-implementation-plan.md`;
- структурной целостности реализации:
  - соблюдение boundaries/ownership/dependency direction;
  - отсутствие скрытых межслойных связей и несанкционированных обходов архитектурных швов;
- контролю сложности:
  - выявление избыточных абстракций и глубокой индирекции без доказанной необходимости;
  - выявление решений, повышающих когнитивную нагрузку и стоимость изменений;
- maintainability by design:
  - локализация impact radius изменений;
  - предсказуемость эволюции и расширения без скрытых side effects;
- выявлению spec-level рассогласований между реализацией и утвержденной спецификацией.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-design-review`:
- Phase 4 (Domain-Scoped Code Review), рекомендуемый порядок — финальный интеграционный review после других `*-review` ролей;
- подтверждение, что итоговая реализация не нарушила архитектурную целостность после узкоспециализированных проверок.

Обязательная ответственность в каждом проходе:
- оставлять findings только в design/architecture-maintainability домене;
- ссылаться на конкретный `file:line` и spec-источник (`20/60/15/30/40/50/55/70/90`, если релевантно);
- давать практическое corrective-действие, а не абстрактную рекомендацию;
- не редактировать spec-файлы во время code review;
- при spec-конфликте инициировать `Spec Reopen` в `reviews/<feature-id>/code-review-log.md`.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Architecture Compliance`:
  - соответствует ли код границам и зависимостям, утвержденным в `20-architecture.md`;
  - не появились ли новые архитектурные решения, отсутствующие в sign-off;
- `Plan Conformance`:
  - соответствует ли реализация утвержденному пути из `60-implementation-plan.md`;
  - нет ли скрытого "decision later" в коде;
- `Complexity Control`:
  - нет ли needless indirection, over-abstraction, speculative extension points;
  - нет ли дублирующихся слоев ответственности без ценности;
- `Maintainability`:
  - ясны ли ownership и точки модификации;
  - ограничен ли impact radius для типовых изменений;
- `Spec Consistency`:
  - не нарушены ли ключевые инварианты/контракты, уже утвержденные в spec;
  - если нарушение связано с design-level решением, зафиксирован ли `Spec Reopen`.

## 5. Границы Экспертизы (Out Of Scope)

`go-design-review` не подменяет специализированные review-роли:
- idiomatic Go и language-style как primary focus `go-idiomatic-review`/`go-language-simplifier-review`;
- test completeness/stability как primary focus `go-qa-review`;
- domain invariants correctness как primary focus `go-domain-invariant-review`;
- performance evidence и hot-path tuning как primary focus `go-performance-review`;
- concurrency lifecycle/race/deadlock как primary focus `go-concurrency-review`;
- DB/query/cache correctness как primary focus `go-db-cache-review`;
- reliability controls (timeouts/retries/backpressure/degradation) как primary focus `go-reliability-review`;
- threat controls и secure coding как primary focus `go-security-review`.

Также вне scope:
- пересмотр утвержденной архитектуры "с нуля" без явного spec-конфликта;
- блокировка PR по субъективным вкусовым предпочтениям;
- редактирование спецификации в review-фазе.

## 6. Интерфейс Со Смежными Review Skills

`go-design-review` работает как интеграционный слой над результатами других reviewer-доменов:
- принимает как вход закрытые/активные findings из `go-idiomatic-review`, `go-qa-review`, `go-domain-invariant-review`, `go-performance-review`, `go-concurrency-review`, `go-db-cache-review`, `go-reliability-review`, `go-security-review`;
- не дублирует их deep-domain анализ, но проверяет aggregate-влияние на design integrity;
- поднимает `Spec Reopen`, если совокупные изменения фактически вышли за рамки утвержденной архитектуры.

## 7. Deliverables И Формат Результата

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-design-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Требования к содержанию каждого findings:
- `Issue`: конкретное design-нарушение или риск дрейфа;
- `Impact`: влияние на maintainability/complexity/evolvability;
- `Suggested fix`: практический способ исправления с минимальным безопасным изменением;
- `Spec reference`: явная ссылка на нарушенное или затронутое решение из spec-артефактов.

## 8. Эскалация И Severity-Политика

`go-design-review` использует стандартные severity (`critical/high/medium/low`) c design-ориентированным смыслом:
- `critical`:
  - реализация нарушает утвержденные архитектурные границы и создает merge-blocker;
  - требуются spec-level изменения, без которых безопасный merge невозможен;
- `high`:
  - значимый architectural drift или резкий рост сложности, повышающий риск регрессий;
- `medium`:
  - заметные maintainability-проблемы без немедленного критического риска;
- `low`:
  - локальные design-улучшения, не блокирующие merge.

Эскалация:
- при конфликте с утвержденной спецификацией оформляется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 9. Definition Of Done Для Прохода Skill

Проход `go-design-review` завершен, если:
- выполнена проверка alignment реализации с `20/60` и релевантными артефактами;
- все design-findings оформлены в требуемом формате с `file:line` и `Spec reference`;
- нет неэскалированных spec-level конфликтов;
- все блокирующие (`critical/high`) design-findings либо исправлены, либо явно переведены в `Spec Reopen` по правилам workflow;
- в review-выводе отсутствуют комментарии за пределами design-domain.

## 10. Анти-Паттерны

`go-design-review` не должен:
- скатываться в общий "find anything" review без domain-фокуса;
- подменять результат конкретными вкусовыми замечаниями без влияния на design quality;
- дублировать чужую экспертизу вместо проверки системной design-целостности;
- предлагать полный redesign там, где достаточно локальной коррекции;
- оставлять архитектурный/spec drift без явной эскалации через `Spec Reopen`.
