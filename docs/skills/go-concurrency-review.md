# Skill Spec: `go-concurrency-review` (Domain-Scoped Review)

## 1. Назначение

`go-concurrency-review` — экспертный review-skill по конкурентному поведению Go-кода в Phase 4 (`Domain-Scoped Code Review`) spec-first workflow.

Ценность skill:
- выявляет риски, которые проявляются только при конкурентном исполнении (race, deadlock, leak, зависания);
- подтверждает, что lifecycle goroutines и cancellation/shutdown поведение безопасны для production;
- дает actionable findings в рамках review-домена без подмены соседних ролей.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за concurrency-review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-concurrency-review` принимает решения по:
- корректности lifecycle goroutines:
  - есть ли завершение по `work done`, `channel close`, `ctx.Done()`, или shutdown-сигналу;
  - нет ли fire-and-forget без осознанной и безопасной модели жизни;
- cancellation/deadline propagation в конкурентных потоках;
- корректности orchestration конкурентной группы:
  - оправдано ли использование `errgroup.WithContext`/`WaitGroup`;
  - есть ли bounded concurrency (`SetLimit` или эквивалентный механизм);
- channel discipline:
  - ownership закрытия канала;
  - отсутствие double-close и send-on-closed;
  - отсутствие блокировок без cancellation path;
- shared-state safety:
  - синхронизация mutable shared state;
  - отсутствие unsynchronized map access;
  - корректность критических секций и lock-стратегии;
- leak/deadlock/backpressure рискам в pipelines/worker pools;
- корректности error propagation между goroutines и вызывающим контуром;
- достаточности verification-сигналов для конкурентных изменений (`go test -race`, релевантные тесты).

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-concurrency-review`:
- Phase 4 review при наличии конкурентных изменений;
- проверка, что реализация соответствует approved spec intent и не добавляет скрытых concurrency-рисков перед `Gate G4`.

Обязательная ответственность в каждом проходе:
- оставлять findings только в concurrency-domain;
- ссылаться на конкретный `file:line` и `Spec reference`;
- формулировать practical fix path, а не абстрактный совет;
- не редактировать spec-файлы в review-фазе;
- при spec-level mismatch оформлять `Spec Reopen` в `reviews/<feature-id>/code-review-log.md`.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Goroutine Lifecycle Safety`:
  - каждая goroutine имеет явный completion/cancellation path;
  - нет утечек при раннем выходе downstream или отмене контекста;
- `Cancellation And Deadline Semantics`:
  - блокирующие операции могут завершиться по отмене;
  - контекст запроса корректно прокинут в конкурентную работу;
- `Channel Ownership And Closure`:
  - однозначно определено, кто закрывает канал;
  - нет closure с receiver-side, если это не владелец producer-side;
  - нет сценариев с неопределенным блокированием send/receive;
- `Shared State Synchronization`:
  - shared mutable state защищен синхронизацией или confinement-моделью;
  - критичные race-risk участки не зависят от "удачного" scheduling;
- `Bounded Concurrency And Backpressure`:
  - отсутствует неограниченное порождение goroutines/очередей;
  - есть управляемая емкость и предсказуемое поведение под нагрузкой;
- `Deadlock And Shutdown Safety`:
  - shutdown path разблокирует waits/sends/receives;
  - lock ordering и channel interaction не создают циклических зависимостей;
- `Concurrency Error Propagation`:
  - ошибки concurrent workers наблюдаемы вызывающим кодом;
  - критичные ошибки не теряются и не маскируются;
- `Concurrency Verification Readiness`:
  - для значимых конкурентных изменений есть достаточный тестовый сигнал, включая `-race` где применимо.

## 5. Границы Экспертизы (Out Of Scope)

`go-concurrency-review` не подменяет соседние reviewer-роли:
- не оценивает endpoint business meaning и продуктовую семантику как primary-domain;
- не выполняет полный idiomatic/style review (`go-idiomatic-review`);
- не выполняет primary performance proof (`go-performance-review`), кроме прямых concurrency-регрессий;
- не выполняет primary DB/cache correctness review (`go-db-cache-review`);
- не выполняет primary reliability-политику (`go-reliability-review`), кроме участков, где проблема прямо в concurrent control flow;
- не выполняет primary security review (`go-security-review`);
- не выполняет полный тест-стратегический аудит (`go-qa-review`).

Также вне scope:
- пересмотр архитектуры без явного spec-конфликта;
- редактирование спецификации в review-фазе;
- субъективные замечания без доказуемого concurrency-risk.

## 6. Интерфейс Со Смежными Review Skills

`go-concurrency-review` передает handoff:
- в `go-reliability-review`, если основной риск в timeout/retry/degradation/shutdown policy, а не в низкоуровневой конкурентной механике;
- в `go-performance-review`, если проблема требует benchmark/profile-evidence hot-path уровня;
- в `go-db-cache-review`, если root cause в transaction/query/cache semantics, а concurrency только симптом;
- в `go-security-review`, если concurrent path приводит к authz/data exposure/security последствиям;
- в `go-qa-review`, если главный gap в отсутствии concurrency-ориентированных тестов;
- в `go-design-review`, если concurrency-проблема вызвана архитектурным drift.

Правило интерфейса:
- `go-concurrency-review` фиксирует concurrency-risk и минимальный safe fix,
- но не захватывает глубокую ownership-область другого review-skill.

## 7. Deliverable Формат Для Review-Лога

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-concurrency-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Минимальные требования к finding:
- `Issue`: конкретный concurrency-дефект или риск;
- `Impact`: как это может сломать корректность/стабильность/merge-safety;
- `Suggested fix`: минимально достаточный способ исправления;
- `Spec reference`: ссылка на релевантный spec-артефакт (`55/70/90` и при необходимости `20/30/40`).

## 8. Severity И Эскалация

Severity-интерпретация в concurrency-domain:
- `critical`: подтвержденный риск некорректности/зависания/утечки goroutines, блокирующий безопасный merge;
- `high`: высокая вероятность race/deadlock/leak или неограниченной конкурентности в значимом path;
- `medium`: локальный concurrency-risk с ограниченным blast radius;
- `low`: улучшения читаемости/безопасности без немедленного merge-block.

Эскалация:
- если безопасный fix требует изменения утвержденного spec intent, создается `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 9. Definition Of Done Для Прохода Skill

Проход `go-concurrency-review` завершен, если:
- проверены все измененные конкурентные участки по обязательному scope;
- все `critical/high` findings задокументированы с `file:line`, impact, fix и spec reference;
- нет неэскалированных spec-level конфликтов;
- вывод остается строго в concurrency-domain;
- при отсутствии проблем явно зафиксировано, что concurrency findings не обнаружены.

## 10. Анти-Паттерны

`go-concurrency-review` не должен:
- превращаться в общий code review вне concurrency-фокуса;
- требовать параллелизм там, где последовательный код проще и надежнее;
- предлагать "лечить" race добавлением `sleep`/тайминговых хаков;
- игнорировать shutdown/cancellation path и оценивать только happy-path;
- оставлять потенциальный deadlock/leak без явной фиксации или эскалации.

## 11. Статус Текущего Документа

Этот файл фиксирует `SCOPE` и `RESPONSIBILITIES` для будущего `SKILL.md`.
Инструкции исполнения (`Working Rules`, `Context Intake`, точный output protocol для runtime-ответов) будут оформлены отдельным шагом.
