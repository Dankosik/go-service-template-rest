# Единый Гайд По Проектированию Skills

Этот документ описывает только то, как писать хорошие `SKILL.md`.
Без автоматизации, без валидации, без `tool policy`, без `evals`, без Python/Go-скриптов.
Этот файл является единым источником для проектирования skill:
- структура и формулировки `SKILL.md`;
- динамическая загрузка инструкций из `docs/*`.

## 1. Цель Skill

`Skill` нужен, чтобы задать повторяемое поведение агента в конкретном типе задач.

Хороший `Skill`:
- легко триггерится в нужных запросах;
- не триггерится в похожих, но нерелевантных запросах;
- ведет агента по понятному сценарию;
- дает предсказуемый формат результата.

Ключевое правило:
- в `description` хранится логика активации;
- в теле `SKILL.md` хранится только исполняемая экспертиза.

## 2. Главный Принцип: Один Skill = Один Тип Работы

Не объединяй в один `Skill` разные режимы.

Плохой пример:
- один `Skill` одновременно про архитектуру, ревью, генерацию кода и тестирование.

Хороший пример:
- отдельный `Skill` только для архитектурных спецификаций;
- отдельный `Skill` только для код-ревью;
- отдельный `Skill` только для UX-текста.

## 3. Где Писать Skill

Рабочие `SKILL.md` хранятся в исполняемых директориях:
- `.agents/skills/<skill-name>/SKILL.md`
- `.claude/skills/<skill-name>/SKILL.md`
- `.gemini/skills/<skill-name>/SKILL.md`
- `.github/skills/<skill-name>/SKILL.md`
- `.cursor/skills/<skill-name>/SKILL.md`

`docs/skills/` содержит только документацию:
- гайды;
- спецификации;
- правила написания skills.

## 4. Структура SKILL.md

`SKILL.md` состоит из двух частей:
- YAML frontmatter;
- тело инструкции в Markdown.

### 4.1. Frontmatter

Обязательные поля:
- `name`
- `description`

Рекомендации:
- `name`: lowercase + hyphen-case, короткий и предметный.
- `description`: это главный триггер. В одном абзаце укажи, что делает skill, когда его использовать и когда его не использовать.

### 4.2. Тело Skill

Рекомендуемый минимальный каркас:

1. `# <Skill Title>`
2. `## Purpose`
3. `## Scope And Boundaries`
4. `## Hard Skills` (для domain-critical skills: review/spec/coder/security/reliability/performance/data/observability)
5. `## Working Rules`
6. `## Output Expectations`
7. `## Definition Of Done`
8. `## Anti-Patterns`
9. `## Context Intake (Dynamic Loading)` (если skill использует локальные инструкции из `docs/*`)

В тело `SKILL.md` не включай:
- тесты триггера;
- `Use when/Skip when` блоки;
- инструкции, которые не влияют на исполнение skill.

## 5. Как Писать Каждый Раздел

## Description (frontmatter)

`description` должен содержать только краткий триггер:
- что делает skill;
- когда применять;
- когда не применять.

Этого достаточно для активации.
В теле `SKILL.md` не повторяй триггерные блоки.

## Purpose

Коротко ответь:
- какую задачу решает skill;
- какой результат считается успешным.

## Scope And Boundaries

Явно зафиксируй границы:
- что skill делает;
- что skill не делает.

Это главный способ избежать «расползания» skill в соседние задачи.

## Working Rules

Опиши рабочий порядок шагов.
Пиши в повелительной форме: «Сделай», «Проверь», «Сформулируй», «Зафиксируй».

Хорошая последовательность:
1. Собрать минимальный контекст.
2. Зафиксировать допущения, если не хватает данных.
3. Выполнить основную задачу skill.
4. Проверить внутреннюю согласованность результата.
5. Вернуть итог в ожидаемом формате.

## Hard Skills

Этот раздел обязателен для skill, где нужна устойчивая инженерная глубина (review/spec/coder домены).

Минимальный формат:
- `Mission`
- `Default Posture`
- доменные компетенции (`... Competency`) с конкретными правилами
- `Evidence Threshold` (какой уровень доказательности обязателен)
- `Review Blockers For This Skill` (что блокирует merge/sign-off)

Правила:
- `Hard Skills` должны быть domain-specific, а не общими фразами.
- Каждый пункт должен быть операционализирован: что проверять, что считать нарушением, что считать достаточным доказательством.
- `Working Rules` исполняют skill, но `Hard Skills` определяют качество инженерных решений.

## Context Intake (Dynamic Loading)

Этот раздел нужен, если skill должен подгружать локальные инструкции по задаче.

Обязательные правила:
- загружай минимально достаточный набор документов;
- не загружай целые папки по умолчанию;
- разделяй `Always load` и `Load by trigger`;
- при конфликте правил выбирай более конкретный документ;
- при нехватке данных фиксируй `[assumption]`.

Рекомендуемый алгоритм:
1. Классифицируй задачу по домену.
2. Загрузи базовые документы skill.
3. Добавь trigger-based документы по сигналам задачи.
4. Останови загрузку, как только покрытие задачи полное.
5. Применяй приоритет specific-over-general.
6. Если данных не хватает, работай через `[assumption]`.

Базовый шаблон:

```markdown
## Context Intake (Dynamic Loading)

Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.

Always load:
- <minimal stable files for this skill>

Load by trigger:
- <signal A>: <doc path>
- <signal B>: <doc path>

Conflict resolution:
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
```

## Output Expectations

Опиши требования к итогу:
- формат;
- обязательные секции;
- уровень детализации;
- язык ответа (если важно).

Формулируй требования проверяемо.
Плохо: «дай качественный ответ».
Хорошо: «верни 4 секции: контекст, решение, риски, next steps».

## Definition Of Done

Это финальный критерий готовности.

Пример:
- все обязательные секции присутствуют;
- нет противоречий между секциями;
- вывод не выходит за границы skill;
- допущения явно отмечены.

## Anti-Patterns

Перечисли, что запрещено в рамках skill.

Примеры:
- уход в нерелевантный домен;
- абстрактные советы без конкретных решений;
- противоречивые инструкции;
- подмена цели skill другой задачей.

## 6. Стиль Формулировок

Пиши так, чтобы агенту не приходилось «догадываться».

Правила:
- короткие фразы;
- один пункт = одно требование;
- конкретные глаголы действия;
- минимально достаточная детализация;
- без расплывчатых слов вроде «лучше», «аккуратнее», «примерно».

## 7. Частые Ошибки При Написании Skills

1. Слишком широкая `description`, из-за чего skill триггерится почти всегда.
2. Триггерные правила дублируются в теле skill и засоряют контекст при каждом вызове.
3. Границы не описаны, поэтому агент уходит в соседние роли.
4. Требования к выходу не формализованы, ответы становятся нестабильными.
5. Слишком длинный и теоретический текст без операционных шагов.

## 8. Шаблон Для Нового SKILL.md

```markdown
---
name: your-skill-name
description: "Что делает skill. Use when: ... Skip when: ..."
---

# Your Skill Title

## Purpose
Коротко: задача и ожидаемый результат.

## Scope And Boundaries
In scope:
- ...
- ...

Out of scope:
- ...
- ...

## Hard Skills
### <Domain> Core Instructions

#### Mission
- ...

#### Default Posture
- ...

#### <Domain> Competency
- ...

#### Evidence Threshold
- ...

#### Review Blockers For This Skill
- ...

## Working Rules
1. ...
2. ...
3. ...
4. ...

## Output Expectations
- Формат ответа: ...
- Обязательные секции: ...
- Ограничения: ...

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.

Always load:
- ...

Load by trigger:
- <signal>: <doc path>
- <signal>: <doc path>

Conflict resolution:
- The more specific document is the decisive rule for that topic.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.

## Definition Of Done
- ...
- ...
- ...

## Anti-Patterns
- ...
- ...
- ...
```

## 9. Ручной Чеклист Перед Сохранением

Перед тем как считать `SKILL.md` готовым, проверь:

1. По `description` понятно, когда skill включать и когда не включать.
2. В теле нет дублирования триггера (`Use when/Skip when`, `Fast Trigger Test` и т.п.).
3. Границы (`In scope/Out of scope`) не пересекаются и не противоречат друг другу.
4. В `Working Rules` есть последовательность действий, а не общие слова.
5. `Output Expectations` описывает результат в проверяемом виде.
6. Если skill использует `docs/*`, есть блок `Context Intake (Dynamic Loading)`.
7. В блоке загрузки есть `Always`, `Load by trigger`, conflict resolution, unknowns.
8. `Definition Of Done` можно проверить глазами без интерпретаций.
9. В тексте нет блоков, не относящихся к самому написанию skill.

## 10. Карта Динамической Загрузки Для Этого Репозитория

Используй эту карту при заполнении `Load by trigger`.

### Go
- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/20-go-concurrency.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/llm/go-instructions/40-go-testing-and-quality.md`
- `docs/llm/go-instructions/50-go-public-api-and-docs.md`
- `docs/llm/go-instructions/60-go-performance-and-profiling.md`
- `docs/llm/go-instructions/70-go-review-checklist.md`

### API
- `docs/llm/api/10-rest-api-design.md`
- `docs/llm/api/30-api-cross-cutting-concerns.md`

### Architecture
- `docs/llm/architecture/10-service-boundaries-and-decomposition.md`
- `docs/llm/architecture/20-sync-communication-and-api-style.md`
- `docs/llm/architecture/30-event-driven-and-async-workflows.md`
- `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`

### Data
- `docs/llm/data/10-sql-modeling-and-oltp.md`
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/30-nosql-and-columnar-decision-guide.md`
- `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- `docs/llm/data/50-caching-strategy.md`

### Security
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`

### Operability / Delivery / Platform
- `docs/llm/operability/10-observability-baseline.md`
- `docs/llm/operability/20-sli-slo-alerting-and-runbooks.md`
- `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- `docs/llm/delivery/10-ci-quality-gates.md`
- `docs/llm/platform/10-containerization-and-dockerfile.md`

## 11. Официальные Источники

- OpenAI Codex Skills: https://developers.openai.com/codex/skills/
- OpenAI Codex Customization: https://developers.openai.com/codex/concepts/customization/
- Anthropic Skills Best Practices: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices
- Anthropic Skills Overview: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview
- Claude Code Skills: https://code.claude.com/docs/en/skills
- Microsoft Agent Skills: https://learn.microsoft.com/en-us/agent-framework/agents/skills
