# Skill Spec: `go-security-review` (Domain-Scoped Review)

## 1. Назначение

`go-security-review` — экспертный review-skill по security-корректности Go-кода в Phase 4 (`Domain-Scoped Code Review`) spec-first workflow.

Ценность skill:
- выявляет security-регрессии до merge в реализациях, которые уже прошли фазу spec sign-off;
- проверяет соответствие кода утвержденным security-решениям и fail-closed ожиданиям;
- дает actionable findings в security-домене без захвата смежных review-ролей.

Workflow-контур (`phases`, `gates`, `freeze/reopen`) задается в `docs/spec-first-workflow.md`; этот skill отвечает только за security-review экспертизу в рамках Phase 4.

## 2. Ядро Экспертизы

`go-security-review` принимает решения по:
- корректности trust-boundary enforcement в измененных участках кода;
- корректности `AuthN/AuthZ` и tenant/object-level access control;
- качеству input validation и bounded parsing для untrusted input;
- предотвращению injection-классов (SQL/NoSQL/command/template);
- SSRF-контролям и безопасности outbound HTTP/RPC вызовов;
- path traversal и file/upload безопасности;
- управлению секретами и предотвращению утечек sensitive data в API/errors/logs/traces;
- abuse-resistance контролям (timeouts, limits, bounded concurrency, rate semantics);
- security fail-path поведению (deny-by-default/fail-closed);
- traceability security-обязательств к approved spec artifacts и тестовым обязательствам.

## 3. Ответственность В Spec-First Workflow

Ключевая роль `go-security-review`:
- выполнять domain-scoped security review в Phase 4;
- подтверждать, что реализация не нарушает security intent, утвержденный в `specs/<feature-id>/50-security-observability-devops.md` и `specs/<feature-id>/90-signoff.md`.

Обязательная ответственность в каждом проходе:
- оставлять findings только в security-domain;
- ссылаться на конкретный `file:line` и `Spec reference`;
- формулировать practical fix path, а не абстрактные рекомендации;
- не редактировать spec-файлы в review-фазе;
- при spec-level mismatch инициировать `Spec Reopen` в `reviews/<feature-id>/code-review-log.md`.

## 4. Scope Проверок (Что Проверяет Skill)

Обязательный проверочный scope:
- `Input Validation And Boundary Parsing`:
  - есть ли explicit size limits и strict decode discipline для untrusted input;
  - нет ли parsing путей с silent acceptance unknown/extra data;
- `AuthN/AuthZ And Tenant Isolation`:
  - разделены ли authentication и authorization обязанности;
  - есть ли object-level authorization и tenant-scoping в критичных ветках;
  - нет ли implicit allow/default-allow поведения;
- `Injection And Query Safety`:
  - параметризованы ли запросы к SQL/NoSQL;
  - нет ли shell/command execution с user-influenced input;
  - нет ли template/encoding bypass путей;
- `Outbound Security And SSRF Controls`:
  - у outbound клиентов есть explicit timeout и policy controls;
  - нет ли user-influenced URL вызовов без allowlist/egress checks;
- `Filesystem, Path, And Upload Safety`:
  - защищены ли file-path операции от traversal/symlink-escape;
  - не используются ли user filenames как trusted storage path;
  - есть ли размерные/типовые ограничения загрузок;
- `Secrets And Sensitive Data Handling`:
  - отсутствуют ли утечки секретов/PII в ответах/ошибках/логах/трейсах;
  - соблюдается ли redaction/sanitization policy;
- `Abuse Resistance And Resource Controls`:
  - есть ли bounded limits для expensive operations;
  - нет ли неограниченных ресурсов, открывающих DoS-вектор;
  - корректно ли определены `429/503` и retry-related semantics на security-критичных путях;
- `Security Verification Readiness`:
  - есть ли покрытие критичных negative-path/security сценариев из `70-test-plan.md`.

## 5. Границы Экспертизы (Out Of Scope)

`go-security-review` не подменяет соседние reviewer-роли:
- не выполняет полный idiomatic/style review (`go-idiomatic-review`, `go-language-simplifier-review`);
- не выполняет архитектурный integrity review как primary-domain (`go-design-review`);
- не выполняет deep performance ownership (`go-performance-review`);
- не выполняет primary concurrency механический аудит (`go-concurrency-review`), кроме случаев прямого security-impact;
- не выполняет primary DB/cache correctness review (`go-db-cache-review`);
- не выполняет primary reliability policy review (`go-reliability-review`), кроме fail-open/fail-closed security последствий;
- не выполняет общий test-strategy ownership (`go-qa-review`);
- не выполняет бизнес-инвариант review как primary-domain (`go-domain-invariant-review`).

Также вне scope:
- пересмотр утвержденной спецификации без явного spec-конфликта;
- редактирование spec-артефактов в review-фазе;
- блокирующие замечания без доказуемого security-impact.

## 6. Интерфейс Со Смежными Review Skills

`go-security-review` передает handoff:
- в `go-concurrency-review`, если root cause в race/deadlock/lifecycle, а security-эффект вторичен;
- в `go-reliability-review`, если основной риск связан с timeout/retry/degradation policy, а security-риск производный;
- в `go-db-cache-review`, если уязвимость вызвана DB/query/cache semantics как primary cause;
- в `go-qa-review`, если основной gap в отсутствии требуемых security-тестов;
- в `go-design-review`, если исправление требует архитектурного переосмысления за пределами security-review domain;
- в `go-domain-invariant-review`, если security-fix влияет на доменные state-transition/invariant behavior.

Правило интерфейса:
- `go-security-review` фиксирует security-risk, impact и минимальный safe fix,
- но не захватывает primary ownership другого review-skill.

## 7. Deliverable Формат Для Review-Лога

Primary deliverable:
- записи в `reviews/<feature-id>/code-review-log.md` в формате workflow:

```text
[severity] [go-security-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Минимальные требования к finding:
- `Issue`: конкретный security-дефект/уязвимость/риск;
- `Impact`: реалистичный security-impact (exploitability, blast radius, merge risk);
- `Suggested fix`: минимально достаточное безопасное исправление;
- `Spec reference`: ссылка на релевантный approved spec (`50/90`, при необходимости `30/40/55/70/20`).

## 8. Матрица Документов Для Экспертизы

### 8.1 Always

- `docs/spec-first-workflow.md` (Phase 4, Reviewer Focus Matrix, Review Findings Format, Gate G4)
- `docs/llm/security/10-secure-coding.md`
- `docs/llm/security/20-authn-authz-and-service-identity.md`
- `specs/<feature-id>/50-security-observability-devops.md`
- `specs/<feature-id>/90-signoff.md`
- `reviews/<feature-id>/code-review-log.md` (если есть)

### 8.2 Trigger-Based

- Если security-semantics видны на API boundary:
  - `specs/<feature-id>/30-api-contract.md`
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Если security-risk пересекается с data/consistency/cache:
  - `specs/<feature-id>/40-data-consistency-cache.md`
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/20-sql-access-from-go.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
  - `docs/llm/data/50-caching-strategy.md`
- Если security-risk связан с reliability/failure semantics:
  - `specs/<feature-id>/55-reliability-and-resilience.md`
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- Если требуется проверка test-obligations:
  - `specs/<feature-id>/70-test-plan.md`
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Если риск касается observability/redaction/debug surface:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Если меняется delivery/platform control surface:
  - `docs/llm/delivery/10-ci-quality-gates.md`
  - `docs/llm/platform/10-containerization-and-dockerfile.md`

## 9. Severity И Эскалация

Severity-интерпретация в security-domain:
- `critical`:
  - подтвержденная уязвимость с высокой exploitability или крупным blast radius;
  - broken access control/tenant escape/secret leakage path, блокирующий безопасный merge;
- `high`:
  - существенное нарушение approved security intent с высокой вероятностью инцидента;
- `medium`:
  - локальный security-risk с ограниченным blast radius, требующий исправления;
- `low`:
  - локальное hardening-улучшение без немедленного merge-block.

Эскалация:
- если safe fix требует изменения утвержденного spec intent, инициируется `Spec Reopen`;
- до закрытия `Spec Reopen` merge по `Gate G4` не считается завершенным.

## 10. Definition Of Done Для Прохода Skill

Проход `go-security-review` завершен, если:
- проверены все измененные security-sensitive участки по обязательному scope;
- все `critical/high` findings оформлены с `file:line`, impact, suggested fix и spec reference;
- все выявленные spec-level mismatch либо устранены, либо эскалированы через `Spec Reopen`;
- кросс-доменные риски переданы через handoff профильным reviewer-ролям;
- вывод остается строго в security-domain;
- при отсутствии проблем явно зафиксировано, что security findings не обнаружены.

## 11. Анти-Паттерны

`go-security-review` не должен:
- превращаться в общий код-ревью без security-фокуса;
- давать расплывчатые замечания без threat model и без exploit-impact;
- смешивать `AuthN` и `AuthZ` в один неразличимый комментарий;
- игнорировать negative-path и проверять только happy-path;
- принимать "internal traffic trusted by default" без явного обоснования;
- оставлять потенциальный security-critical дефект без явной фиксации или эскалации.

## 12. Статус Текущего Документа

Этот файл фиксирует `SCOPE` и `RESPONSIBILITIES` для будущего `SKILL.md` по `go-security-review`.
Runtime-инструкция (`Working Rules`, `Context Intake`, output protocol для ответов) будет оформлена отдельным шагом.
