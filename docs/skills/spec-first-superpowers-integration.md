# Интеграция Идей `superpowers` В Наш `spec-first` Workflow

## 1. Контекст Проблемы

Мы обсуждаем, как адаптировать идеи из:
- `using-superpowers`:
  - `https://github.com/obra/superpowers/blob/main/skills/using-superpowers/SKILL.md`
- `brainstorming`:
  - `https://github.com/obra/superpowers/blob/main/skills/brainstorming/SKILL.md`

Под наш процесс:
- `docs/spec-first-workflow.md`
- `AGENTS.md`

Ключевой запрос:
1. Хотим, чтобы каждое сообщение в чате проходило через контролируемый workflow.
2. Хотим deterministic routing: какой skill вызывать на каждом turn.
3. Хотим использовать `brainstorming` как обязательный pre-step перед спецификацией.
4. Принимаем, что ради этого можно переписать/усилить инструкции.

## 2. Что Уже Есть В Репозитории

1. Спецификация фаз и gate-модели (`G0..G4`) уже формализована в `docs/spec-first-workflow.md`.
2. Навигация по skill-классам уже есть:
   - `*-spec` для фазы спецификации,
   - `go-coder` / `go-qa-tester` для реализации,
   - `*-review` для review-фазы.
3. В `AGENTS.md` уже есть динамическая загрузка skills, но нет message-level обязательного gate перед каждым ответом.

## 3. Ограничения Платформы (Важно)

По контексту обсуждения:
1. В Codex нет гарантированного механизма `hooks`, эквивалентного "жесткому pre-message hook" как отдельной встроенной фиче.
2. Поэтому "100% enforcement" только через prompt-текстом не гарантируется как hard runtime contract.
3. Реалистичный подход:
   - `Prompt Policy` (инструкции + skill-router),
   - при необходимости `Runtime Policy` (внешний оркестратор/валидатор turn-ов).

## 4. Что Берем Из `superpowers`, А Что Адаптируем

### 4.1 Keep

1. Принцип: если есть даже слабый сигнал применимости skill, skill должен быть вызван до действий.
2. Идея process-skill как первого шага на turn.
3. Явный порядок приоритетов skill-классов.

### 4.2 Adapt

1. Вместо generic terminal state (`writing-plans`) используем наш spec-first lifecycle.
2. Вместо `docs/plans/*` используем артефакты `specs/<feature-id>/*`.
3. Routing строим по нашим фазам (`Phase 0/1/2/2.5/3/4/5`) и нашим skill-ролям.
4. Добавляем исключения для задач, где brainstorming не нужен (bugfix, review, точечный вопрос).

### 4.3 Drop

1. Любые шаги, которые конфликтуют с `Spec Freeze`, `Spec Clarification Request`, `Spec Reopen`.
2. Generic процессы, не привязанные к нашим gates и owner-модели.

## 5. Предлагаемая Архитектура Управления Turn-ами

## 5.1 Message Gate `M0` (Новый Обязательный Шаг)

Перед каждым ответом агент обязан выполнить `M0`:
1. Определить тип запроса (`intent`).
2. Определить текущую фазу workflow (`phase`).
3. Определить список кандидатов skills.
4. Выбрать skill(ы) по матрице `phase x intent`.
5. Зафиксировать короткое обоснование выбора.
6. Только после этого выполнять действия/отвечать.

## 5.2 Рекомендуемая Классификация Intent

1. `new_feature_or_behavior_change`
2. `spec_enrichment`
3. `implementation`
4. `test_implementation`
5. `code_review`
6. `bug_or_failing_test`
7. `informational_question`
8. `workflow_meta_question`

## 5.3 Routing Matrix (Базовая Версия)

1. `new_feature_or_behavior_change`:
   - сначала `spec-first-brainstorming`,
   - затем `go-architect-spec` и запуск Phase 0/1.
2. `spec_enrichment`:
   - skill-петля Phase 2 из `docs/spec-first-workflow.md`.
3. `implementation`:
   - `go-coder` (после G2.5).
4. `test_implementation`:
   - `go-qa-tester`.
5. `code_review`:
   - соответствующие `*-review` skills по доменной области.
6. `bug_or_failing_test`:
   - `go-systematic-debugging` как process-skill first.
7. `informational_question`:
   - lightweight path без тяжелой skill-цепочки, если не происходит изменение артефактов/кода.
8. `workflow_meta_question`:
   - `go-architect-spec` (как owner workflow-coherence) или отдельный process-skill для governance.

## 6. Новые Skills Для Интеграции

## 6.1 `using-spec-first-superpowers` (Process-Skill)

Назначение:
1. Обязательный pre-turn роутер.
2. Реализует `M0` и правила выбора skills.
3. Применяет правило "если есть даже минимальный шанс применимости, проверяй skill".

Обязательные части skill:
1. Алгоритм определения `phase`.
2. Алгоритм определения `intent`.
3. Таблица выбора `required` и `optional` skills.
4. Stop-условия, когда можно отвечать без тяжелого skill-процесса.
5. Anti-rationalization red flags (адаптированные под наш репозиторий).

## 6.2 `spec-first-brainstorming` (Process-Skill Перед Spec Phase)

Назначение:
1. Структурировать problem framing до запуска `*-spec` команды.
2. Убрать размытые формулировки до старта Phase 0.
3. Подготовить качественный вход в `00/10/80` артефакты.

Выходы skill:
1. Четко зафиксированный problem statement.
2. Scope/Non-goals/constraints.
3. Набор исходных assumptions.
4. Стартовый список open questions.
5. Явное решение: можно ли переходить в Phase 0.

Terminal handoff:
1. `spec-first-brainstorming` завершен.
2. Далее запускается `go-architect-spec` для formal spec initialization.

## 7. Встраивание В Текущий `spec-first-workflow.md`

Рекомендуемые изменения в workflow-док:
1. Добавить `Phase -1: Brainstorming And Problem Framing` перед текущей `Phase 0`.
2. Добавить `Gate B0`:
   - проблема нормализована,
   - scope/non-goals согласованы,
   - критические assumptions зафиксированы,
   - стартовые open questions созданы.
3. После `Gate B0` разрешен вход в текущую `Phase 0`.

Рекомендуемые изменения в `AGENTS.md`:
1. Добавить обязательный `M0` pre-turn gate.
2. Добавить правило: если запрос похож на новый feature/refactor/behavior change, сначала `spec-first-brainstorming`.
3. Оставить принцип минимальной загрузки контекста, но сделать process-skill обязательным.

## 8. Нужен Ли Runtime Enforcement

Если достаточно soft enforcement:
1. Хватает обновленных инструкций + process-skills.

Если нужен hard enforcement:
1. Добавить внешний wrapper/оркестратор с проверкой:
   - был ли выполнен `M0`,
   - верно ли выбран skill для текущей фазы/intent,
   - разрешено ли действие при текущем gate-state.
2. Без успешной проверки turn не принимается как валидный.

## 9. Rollout План (Прагматичный)

1. Создать `using-spec-first-superpowers` skill.
2. Создать `spec-first-brainstorming` skill.
3. Обновить `AGENTS.md` (`M0`, routing rules).
4. Обновить `docs/spec-first-workflow.md` (Phase -1 + Gate B0).
5. Прогнать на 3-5 feature запросах и собрать incidents misrouting.
6. Подкрутить routing matrix по фактическим ошибкам.
7. При необходимости добавить внешний runtime validator.

## 10. Риски И Контрмеры

1. Риск: over-triggering skills и рост latency.
   - Контрмера: четкий lightweight-path для informational/meta вопросов.
2. Риск: процессный skill начнет заменять domain expertise.
   - Контрмера: жестко оставить роль process-skill только как router/gate.
3. Риск: конфликт с правилом минимальной загрузки контекста.
   - Контрмера: всегда грузить только router-skill + целевые доменные skills по матрице.
4. Риск: "ложное чувство 100% контроля" только за счет prompt.
   - Контрмера: для hard guarantees добавить runtime validation слой.

## 11. Критерии Успеха Интеграции

1. На каждом turn фиксируется выбранный skill и reason.
2. Для feature-задач brainstorming стабильно выполняется до spec-init.
3. Количество spec-clarification пауз в Phase 3 снижается.
4. Снижается число late-stage `Spec Reopen` из-за невыявленных входных неопределенностей.
5. Нет деградации скорости на lightweight informational turn-ах.

## 12. Открытые Вопросы Для Следующего Шага

1. Нужен ли отдельный файл-манифест routing matrix в YAML/Markdown?
2. Нужен ли единый audit-log формат для turn-routing?
3. Нужен ли внешний runtime validator уже сейчас или после пилота?
4. Какие intents считаем "lightweight path" в v1 без обязательного brainstorming?

## 13. Резюме

Интеграция `brainstorming` и идеи `using-superpowers` в наш `spec-first` workflow практична и полезна, если:
1. сделать `M0` обязательным pre-turn gate;
2. добавить `spec-first-brainstorming` как Phase -1 перед спецификацией;
3. использовать явный routing по `phase x intent`;
4. разделить soft prompt-level контроль и hard runtime-level контроль.

Это даст предсказуемый вход в спецификацию, снизит хаос на ранних этапах и сохранит текущую сильную gate-модель `G0..G4`.
