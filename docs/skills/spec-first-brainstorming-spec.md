# Спецификация Skill `spec-first-brainstorming`

## 1. Цель

`spec-first-brainstorming` — process-skill для фазы до спецификации.

Его задача:
1. Превратить исходный запрос в структурированный вход для spec-first процесса.
2. Снять раннюю неопределенность до запуска `*-spec` команды.
3. Подготовить артефакты, достаточные для входа в `Phase 0` из `docs/spec-first-workflow.md`.

Итогом работы skill должен быть `Gate B0` (предлагаемый pre-gate) с явным `pass/fail`.

## 2. Позиция В Workflow

Место skill в цепочке:
1. `M0` routing (`using-spec-first-superpowers`) определяет intent `new_feature_or_behavior_change`.
2. Запускается `spec-first-brainstorming`.
3. После успешного `B0` handoff в `go-architect-spec` и старт `Phase 0`.

Skill не заменяет ни один `*-spec` skill и не принимает финальные архитектурные/контрактные решения.

## 3. Ответственность Skill

Skill отвечает за:
1. Нормализацию проблемы и формулировки задачи.
2. Фиксацию scope/non-goals/constraints.
3. Первичный реестр assumptions и неизвестных.
4. Формирование стартового `open questions` backlog.
5. Проверку готовности входа в спецификацию (`B0`).

Skill не отвечает за:
1. Полноценную архитектуру (`20-architecture.md`).
2. API/data/security/reliability design decisions уровня sign-off.
3. Реализацию кода и тестов.
4. Доменный code review.

## 4. Scope And Boundaries

In scope:
1. Feature framing и problem decomposition.
2. Scope control и фиксация non-goals.
3. Выявление рисков и информационных пробелов.
4. Подготовка черновых входов в `specs/<feature-id>/00/10/80`.
5. Явное решение: можно переходить в `Phase 0` или нужен дополнительный discovery.

Out of scope:
1. Спор о конкретной реализации до завершения framing.
2. Детализация endpoint payload contracts.
3. Физическое data modeling/migrations.
4. Security/observability/devops hardening design.
5. Любые решения, которые должны закрываться `*-spec` ролями на `Phase 1/2`.

## 5. Trigger Rules (Для Будущего `description`)

Use when:
1. Пользователь инициирует новую фичу/рефакторинг/изменение поведения.
2. Запрос расплывчатый и требует структурирования до спец-процесса.
3. Нужен согласованный problem statement перед запуском spec-loop.

Skip when:
1. Это bugfix с активным дефектом (`go-systematic-debugging`).
2. Это code review (`*-review` skills).
3. Это чисто implementation по уже готовому `65-coder-detailed-plan.md`.
4. Это точечный informational вопрос без изменения артефактов.

## 6. Входы И Зависимости

Минимальные входы:
1. Последний пользовательский запрос.
2. `docs/spec-first-workflow.md` (Phase model + gates).
3. `AGENTS.md` (repo contract и execution loop).
4. Если есть: существующие артефакты `specs/<feature-id>/`.

Опциональные входы:
1. Связанные issue/ADR/PRD.
2. Текущие ограничения delivery/platform/security из `docs/llm/*` по trigger-правилам.

Правило контекста:
1. Загружать только минимально достаточный набор документов.
2. Не загружать целые папки без сигнала.

## 7. Обязательные Выходы

Skill обязан произвести:
1. `Problem Frame Summary`:
   - problem statement;
   - business/user impact;
   - success criteria.
2. `Scope Frame`:
   - in-scope;
   - out-of-scope;
   - constraints.
3. `Assumptions Register`:
   - `[assumption]` + риск + способ валидации.
4. `Open Questions Seed`:
   - список вопросов с owner и unblock condition.
5. `B0 Decision`:
   - `pass` или `fail`;
   - причины;
   - что нужно, чтобы получить `pass`.

Если запрос оформляется в feature-папке, минимум должен быть отражен в:
1. `specs/<feature-id>/00-input.md`
2. `specs/<feature-id>/10-context-goals-nongoals.md`
3. `specs/<feature-id>/80-open-questions.md`

## 8. Рабочий Алгоритм

1. Классифицировать запрос: действительно ли это `new_feature_or_behavior_change`.
2. Нормализовать исходный запрос в 1 короткий problem statement.
3. Явно выделить goal, non-goals и constraints.
4. Зафиксировать известные unknowns и предположения.
5. Проверить, нет ли ранних конфликтов с текущими repo-инвариантами.
6. Сформировать стартовый список open questions с приоритизацией.
7. Проверить критерии `B0`.
8. Вернуть structured handoff для `go-architect-spec`.

## 9. Gate `B0` (Pre-Spec Readiness)

`B0 pass` только если:
1. Проблема и ожидаемое изменение сформулированы однозначно.
2. Scope и non-goals зафиксированы без конфликтов.
3. Критические assumptions явно отмечены.
4. Открытые вопросы перечислены и приоритизированы.
5. Нет скрытого перехода к архитектурным решениям, которые должны быть закрыты `*-spec` ролями.

`B0 fail` если:
1. Остается неоднозначность в целях/границах.
2. Критические ограничения неизвестны и не зафиксированы.
3. Вопросы есть, но нет owner/unblock condition.

## 10. Handoff Contract

При `B0 pass` skill обязан передать:
1. Нормализованный вход для `Phase 0`.
2. Список первоочередных вопросов для `go-architect-spec`.
3. Явный статус: `Ready for Phase 0`.

При `B0 fail` skill обязан передать:
1. Причины отказа.
2. Минимальный набор данных/решений, требуемых для повторного запуска.
3. Явный статус: `Blocked before Phase 0`.

## 11. Output Expectations

Рекомендуемый формат ответа skill:

```text
Problem
Scope
Constraints
Assumptions
Open Questions
B0 Decision
Handoff
```

Требования:
1. Формулировки короткие и проверяемые.
2. Нет архитектурных "решений заранее".
3. Все assumptions помечены как `[assumption]`.

## 12. Definition Of Done

Skill считается выполненным, если:
1. Все обязательные выходы (раздел 7) сформированы.
2. Вынесено однозначное `B0 pass/fail` решение.
3. Подготовлен handoff в `go-architect-spec`.
4. Нет выхода за границы process-skill в domain design.

## 13. Anti-Patterns

1. Переход к архитектуре/контрактам вместо framing.
2. Общие фразы без конкретных ограничений и вопросов.
3. "Вопросы на потом" без owner/unblock condition.
4. Смешивание bugfix/debug и feature brainstorming.
5. Выдача `B0 pass` при незафиксированной критической неопределенности.

## 14. Черновик Frontmatter Для Будущего `SKILL.md`

```yaml
---
name: spec-first-brainstorming
description: "Structure and de-risk feature requests before spec design in this repository's spec-first workflow. Use when starting new feature/refactor/behavior-change work and you need a clear problem frame, scope/non-goals, assumptions, and prioritized open questions before Phase 0. Skip when the task is active bug debugging, code review, or implementation on an already approved coder plan."
---
```

## 15. Минимальный План Реализации Skill

1. Создать runnable skill:
   - `skills/spec-first-brainstorming/SKILL.md`
2. Синхронизировать mirrors:
   - `make skills-sync`
   - `make skills-check`
3. Добавить routing reference в process-skill `using-spec-first-superpowers`.
4. После этого обновить `docs/spec-first-workflow.md`:
   - добавить `Phase -1` и `Gate B0`.
