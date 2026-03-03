# Спецификация Skill `using-spec-first-superpowers`

## 1. Цель

`using-spec-first-superpowers` — process-skill для обязательного pre-turn контроля.

Его задача:
1. До любого действия или ответа выполнять message-gate `M0`.
2. Детерминированно выбирать, какие skill(ы) запускать на текущем turn.
3. Не допускать действий вне допустимой фазы `spec-first` workflow.

Итогом работы skill на каждом turn должен быть явный routing result:
1. `route_pass`
2. `route_lightweight`
3. `route_blocked`

## 2. Позиция В Workflow

Место skill в цепочке:
1. Пользовательский запрос поступает в чат.
2. `using-spec-first-superpowers` выполняет `M0`.
3. После этого запускаются выбранные доменные/process skills.
4. Только после выполнения routing-разрешения агент отвечает или вносит изменения.

Skill является orchestration-layer и не заменяет экспертизу `*-spec`, `go-coder`, `go-qa-tester`, `*-review`.

## 3. Ответственность Skill

Skill отвечает за:
1. Классификацию запроса по `intent`.
2. Определение текущей workflow-фазы (`phase`) и gate-state.
3. Выбор `required` и `optional` skills по матрице `phase x intent`.
4. Применение правила "даже минимальная вероятность применимости skill -> skill-кандидат".
5. Явную фиксацию routing-решения и причин.
6. Блокировку действий при фазовых нарушениях (`Spec Freeze`, gate violations).

Skill не отвечает за:
1. Архитектурные/контрактные/доменные решения предметной области.
2. Реализацию кода и тестов.
3. Проведение code review.
4. Подмену downstream skill-результатов своими выводами.

## 4. Scope And Boundaries

In scope:
1. Pre-turn routing policy.
2. Intent/phase/gate classification.
3. Skill selection priority and execution order.
4. Lightweight-path decision для informational turn-ов.
5. Block/allow решение для текущего действия.

Out of scope:
1. Предметная инженерная работа, принадлежащая downstream skills.
2. Редактирование спецификаций/кода без выбранного skill-контекста.
3. Любой "скрытый" bypass `M0`.

## 5. Trigger Rules (Для Будущего `description`)

Use when:
1. Любой новый пользовательский turn в этом репозитории.
2. Нужно определить, какой skill запускать первым.
3. Требуется контроль соответствия текущей workflow-фазе.

Skip when:
1. Только если routing уже выполнен на этом turn и зафиксирован.
2. Нет skip для "простых" вопросов: даже в этом случае нужен `M0`, но результат может быть `route_lightweight`.

## 6. Входы И Зависимости

Минимальные входы:
1. Текущий пользовательский запрос.
2. История текущего треда (минимум последняя активная задача/фаза).
3. `docs/spec-first-workflow.md`.
4. `AGENTS.md`.
5. Реестр доступных skills (`skills/*` + mirrors).

Опциональные входы:
1. Артефакты `specs/<feature-id>/*`, если задача уже в feature-контуре.
2. `reviews/<feature-id>/code-review-log.md`, если идет review/reopen контур.
3. `docs/skills/spec-first-superpowers-integration.md`.

Правило контекста:
1. Загружать минимально достаточный набор.
2. Не загружать все skills полностью.
3. Грузить тело skill только после выбора кандидата.

## 7. Обязательные Выходы

Skill обязан формировать `Routing Record`:
1. `intent`
2. `phase`
3. `gate_state`
4. `required_skills`
5. `optional_skills`
6. `selected_order`
7. `decision` (`route_pass` / `route_lightweight` / `route_blocked`)
8. `reason`
9. `constraints` (например `Spec Freeze`)
10. `next_action`

Если `route_blocked`, обязательно:
1. указать причину блокировки;
2. указать минимальное условие разблокировки.

## 8. Рабочий Алгоритм `M0`

1. Определить, есть ли активный feature-контур и текущая фаза.
2. Классифицировать intent текущего запроса.
3. Проверить gate-ограничения для этой фазы.
4. Построить candidate list skills по `phase x intent`.
5. Применить правило минимальной вероятности применимости (1%-правило).
6. Назначить `required` skills.
7. Назначить `optional` skills.
8. Отсортировать execution order по приоритетам.
9. Сформировать `Routing Record`.
10. Разрешить действие (`route_pass` / `route_lightweight`) или заблокировать (`route_blocked`).

## 9. Определение Фазы И Gate-State

Фаза определяется по артефактам и контексту:
1. `Phase -1` (если внедрен) — pre-spec brainstorming.
2. `Phase 0/1/2` — спецификация.
3. `Phase 2.5` — detailed coder plan.
4. `Phase 3` — implementation.
5. `Phase 4` — domain review.
6. `Phase 5` — merge/post-fact.

Gate-state учитывает минимум:
1. `G2`/`G2.5` для допуска к coding.
2. `Spec Freeze` после `G2`.
3. `Spec Clarification Request` / `Spec Reopen` блокировки.

Если phase или gate-state не определимы:
1. зафиксировать `[assumption]`;
2. выбрать самый безопасный путь;
3. при высоком риске вернуть `route_blocked`.

## 10. Taxonomy Intent

Базовые intents:
1. `new_feature_or_behavior_change`
2. `spec_enrichment`
3. `implementation`
4. `test_implementation`
5. `code_review`
6. `bug_or_failing_test`
7. `informational_question`
8. `workflow_meta_question`

Правило конфликтов intent:
1. Использовать наиболее risk-heavy intent.
2. При равенстве приоритета выбрать более строгий process path.

## 11. Политика Выбора Skills

Приоритет классов:
1. Process/safety skills.
2. Phase-mandatory skills.
3. Domain skills.
4. Optional refinement skills.

Базовые соответствия:
1. `new_feature_or_behavior_change`:
   - `required`: `spec-first-brainstorming`
   - `next`: `go-architect-spec`
2. `spec_enrichment`:
   - `required`: `go-architect-spec`
   - `optional`: релевантные `*-spec` по домену изменения
3. `implementation`:
   - `required`: `go-coder`
   - guard: только при `G2.5 pass`
4. `test_implementation`:
   - `required`: `go-qa-tester`
5. `code_review`:
   - `required`: один или несколько `*-review` skills по измененному домену
6. `bug_or_failing_test`:
   - `required`: `go-systematic-debugging`
7. `informational_question`:
   - допускается `route_lightweight` без тяжелого skill-chain
8. `workflow_meta_question`:
   - `required`: `go-architect-spec` или governance-oriented process path

## 12. Правила Блокировки И Эскалации

`route_blocked` обязателен, если:
1. coding запрошен без `G2.5`.
2. попытка изменить spec в `Spec Freeze` без `Spec Reopen`.
3. review-действие запрошено вне review-пути или не тем классом skills.
4. риск изменения контракта/архитектуры без возврата в spec-phase.

Эскалации:
1. `Spec Clarification Request` — при ambiguity в `Phase 3`.
2. `Spec Reopen` — при spec mismatch в review.

## 13. Output Expectations

Рекомендуемый формат ответа skill:

```text
Intent
Phase
Gate State
Routing Record
Decision
Constraints
Next Action
```

Требования:
1. `Decision` всегда явный.
2. `required_skills` перечислены явно.
3. `selected_order` перечислен явно.
4. При `route_lightweight` указано, почему heavy-chain не нужен.
5. При `route_blocked` указано, как перейти в `route_pass`.

## 14. Definition Of Done

Skill считается выполненным на turn, если:
1. Выполнен `M0`.
2. Сформирован полный `Routing Record`.
3. Принято однозначное решение (`pass/lightweight/blocked`).
4. Нет действий в обход выбранного routing.

## 15. Anti-Patterns

1. Ответ или действие до выполнения `M0`.
2. "Простой вопрос, можно пропустить routing".
3. Выбор skills без фиксации причины.
4. Множественные skills без приоритета и порядка.
5. Игнорирование gate-state (`Spec Freeze`, `G2.5`).
6. Подмена domain skills оркестрационным skill.
7. `route_lightweight` для запроса, который реально меняет артефакты/код.

## 16. Ограничение Enforcement

На уровне инструкций skill обеспечивает strong policy, но не абсолютный runtime guarantee.

Для hard enforcement потребуется внешний validator/оркестратор, который:
1. проверяет факт выполнения `M0`;
2. валидирует корректность selected skills;
3. блокирует turn при policy-нарушениях.

## 17. Черновик Frontmatter Для Будущего `SKILL.md`

```yaml
---
name: using-spec-first-superpowers
description: "Run mandatory pre-turn routing for this repository's spec-first workflow. Use on every user message to classify intent and phase, select required skills, enforce gate constraints, and decide route_pass/route_lightweight/route_blocked before any response or action."
---
```

## 18. Минимальный План Реализации Skill

1. Создать runnable skill:
   - `skills/using-spec-first-superpowers/SKILL.md`
2. Синхронизировать mirrors:
   - `make skills-sync`
   - `make skills-check`
3. Встроить ссылку на этот process-skill в `AGENTS.md` как pre-turn правило.
4. Обновить `docs/spec-first-workflow.md`:
   - добавить явный `M0` message-gate раздел.
5. После пилота обновить routing-matrix по фактическим misrouting кейсам.
