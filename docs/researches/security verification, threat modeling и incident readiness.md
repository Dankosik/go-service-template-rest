# Security verification, threat modeling и incident readiness для production-ready Go-микросервиса

## Scope

Этот стандарт предназначен для production-ready микросервисов на Go, которые: принимают **недоверенные входы** (HTTP/gRPC, очереди, webhooks), работают с **данными/учётками/доступами**, зависят от **внешних сервисов/библиотек**, и должны быть **поддерживаемыми и безопасными** при активном использовании LLM-инструментов в разработке (генерация кода, тестов, инфраструктурных файлов). Основание: необходимость “вшивать” практики безопасной разработки в SDLC, а не добавлять их постфактум, прямо описана в SSDF. citeturn4view1turn6view0

Под “security verification” здесь понимается не один инструмент, а **система обязательных артефактов и проверок** (threat model + review + автоматические тесты/сканы + политика уязвимостей + готовность к инцидентам), которая выполняется **на каждом изменении** и на релизе в соответствии с риском. Это согласуется с SSDF (PW.7 code review/analysis, PW.8 тестирование исполняемого кода, включая примеры fuzzing, и RV.* для процесса обработки уязвимостей). citeturn7view3turn9view0turn9view2turn17view0

Подход **не стоит применять “целиком”** (или стоит применять “облегчённый профиль”), если вы делаете: одноразовый прототип, учебный проект, локальную утилиту без сети/данных/пользователей, либо PoC “на выброс”, где стоимость процесса превышает ценность результата. Тем не менее, даже для прототипа рекомендуется минимум: защита от утечек секретов и регулярное сканирование зависимостей — это снижает риск случайных утечек и попадания известных уязвимостей. citeturn2search4turn11view0turn13view1

## Recommended defaults для greenfield template

Рекомендуемые “boring defaults” ниже сформулированы так, чтобы LLM не “догадывалась”, а имела **явный контракт**: какие артефакты существуют, какие проверки обязательны, и что считается Definition of Done.

**Базовая рамка процесса (рекомендуемый каркас):**
- В качестве “общего словаря” практик и привязки к SDLC используйте SSDF от entity["organization","NIST","us standards agency"]: он прямо говорит, что secure-практики нужно добавлять в SDLC, включает threat/risk modeling (PW.1.1), code review/analysis (PW.7) и security testing (PW.8), а также процесс реагирования на уязвимости (RV.*). citeturn4view1turn7view2turn7view3turn9view0turn17view0
- Для threat modeling и связки угроз ↔ мер ↔ тестов используйте cheat sheets от entity["organization","OWASP","web app security nonprofit"] как практический минимум: 4 шага (decomposition → threat identification/ranking → mitigations → review/validation) + поддержка DFD и trust boundaries как базового артефакта. citeturn20view0turn1search5
- Для “repo-level” норм (ветка защищена, секреты не попадают в git, политики SAST/SCA, уязвимости публикуются и обрабатываются) используйте OpenSSF OSPS Baseline как готовый набор контролей и формулировок MUST. Он явно требует предотвращать хранение незашифрованных секретов в VCS и оформлять политики SAST/SCA с gate’ами. citeturn4view0turn5view1

**Пайплайн verification для Go по умолчанию:**
- Govulncheck как “низкошумный” анализ уязвимых зависимостей для Go, который показывает, есть ли **реальный reachable impact** через call stacks. Это “boring default” именно для Go-экосистемы. citeturn11view0turn13view1turn0search3turn0search7
- Встроенный fuzzing Go (“go test -fuzz=…”) как обязательная техника для критичных парсеров/валидаторов/преобразований входных данных; поддержка и ограничения прямо описаны в официальной документации. citeturn11view0turn14view0turn9view2

**Секреты и supply chain hygiene по умолчанию:**
- Если репозиторий на entity["company","GitHub","code hosting platform"]: включайте secret scanning + push protection (как минимум на репозитории/организации) и используйте Dependabot для security updates и version updates. citeturn2search4turn2search11turn2search1
- Для “portable CI” (не завязано на конкретный хостинг) добавьте независимый secrets scanner (например, Gitleaks) как job в CI, чтобы ловить секреты не только через platform features. citeturn15search0turn4view0turn5view1

**Audit logging и incident readiness по умолчанию:**
- Логи должны поддерживать расследования и реакцию: OWASP указывает, что application logging должен включаться для security events, быть консистентным и пригодным для обработки системами log management; OWASP Top 10 отдельно подчёркивает, что без логирования auditable events и мониторинга обнаружение/эскалация инцидентов невозможны. citeturn19view0turn21view0
- Для облачно-нативного контекста entity["organization","Cloud Native Computing Foundation (CNCF)","linux foundation cloud native"] подчёркивает критичность “actionable audit events” и немедленной пересылки логов в место, недоступное злоумышленнику из компрометированного кластера/учётки. citeturn16view0
- Incident response и “continuous improvement” берите из NIST 800-61r3: он рекомендует интегрировать IR в риск-менеджмент, предлагает модель IR через функции CSF 2.0 и прямо ссылается на необходимость упражнений (exercises/tabletop) с отсылкой на NIST 800-84. citeturn10view0turn10view2turn8search4

## Decision matrix / trade-offs

Таблица ниже — “decision matrix” для template. В колонке Default — рекомендуемый boring вариант, в колонке Trade-offs — когда и почему менять.

| Решение | Default для template | Альтернативы | Trade-offs / когда менять |
|---|---|---|---|
| Модель secure SDLC | SSDF как “каркас практик” (PW.* + RV.*) citeturn6view0turn7view3turn17view0 | SAMM как maturity model citeturn12search1turn12search21 | SSDF проще “приземлить” на конкретные gates и артефакты; SAMM полезнее для оценки зрелости и дорожной карты, но тяжелее как “шаблон на старт”. citeturn6view0turn12search1 |
| Threat modeling формат | DFD + trust boundaries + список угроз/мер/проверок (4 шага OWASP) citeturn20view0turn1search5 | Формальные методики (PASTA/OCTAVE) citeturn20view0 | Формализм даёт глубину, но обычно дороже по времени; для greenfield microservice чаще достаточно DFD+STRIDE-прохода и поддержания модели как living-doc. citeturn20view0turn7view2 |
| Abuse cases как артефакт | Использовать как опциональный слой: “abuse cases → security tests”, но держать lightweight citeturn20view1 | Не делать abuse cases, ограничиться “security tests checklist” | OWASP отмечает, что abuse cases “редко используются” и могут быть heavyweight; зато они хорошо связывают угрозы с тест-кейсами и acceptance criteria. Рекомендуется включить шаблон и правила, но не превращать в бюрократию. citeturn20view1turn18view0 |
| SAST | CodeQL (default или advanced setup) citeturn22view1 | gosec/semgrep и др. (по необходимости) | CodeQL хорошо интегрируется как code scanning и имеет обновляемые query suites; другие инструменты могут ловить специфические паттерны/ошибки, но увеличивают шум и операционные издержки. Для template важнее стабильность и минимум “лишних догадок”. citeturn22view1turn5view1 |
| SCA / dependency vulnerabilities | govulncheck как Go-first “low-noise” сканер citeturn11view0turn13view1 | Общие SCA/сканеры (например, Trivy) citeturn15search2turn15search5 | Govulncheck снижает шум за счёт reachability анализа, но покрывает только известные vuln по базе Go; общий сканер полезен для контейнеров/ОС-пакетов, но может быть шумнее. Практика: govulncheck на PR, container/image scan на release. citeturn13view1turn15search5 |
| Secrets предупреждение | Push protection + secret scanning на платформе citeturn2search4turn2search17 | Доп. CI job с gitleaks/trufflehog citeturn15search0turn15search4 | Платформенный push protection предотвращает утечки “на входе”, но зависит от платформы. CI-сканер даёт переносимость и дополнительный слой, но требует allowlist/настройки. citeturn2search4turn15search0 |
| Fuzzing gate | Короткий fuzz smoke на PR + долгий fuzz по расписанию/на release citeturn14view0turn9view2 | Только ночной fuzz / вообще без fuzz | Fuzzing эффективен для edge cases и security issues, но может быть дорог по времени; официальные доки описывают запуск через `go test -fuzz` и ограничения платформ. Поэтому “двухскоростной” режим — типичный компромисс. citeturn14view0turn11view0 |
| Incident readiness | Минимум: runbooks + audit events + exercises/tabletop по расписанию citeturn10view2turn16view0turn19view0 | “Реакция по факту” | NIST связывает IR с подготовкой и continuous improvement, а CNCF подчёркивает роль audit events для incident response. Таблетопы (NIST 800-84) — дешёвый способ проверить готовность до реального инцидента. citeturn10view0turn8search4turn16view0 |

## Security verification, threat modeling и incident readiness в lifecycle template

Ниже — практический “security process document” для шаблона репозитория. Он рассчитан на то, что LLM будет **генерировать не только код**, но и дополнять threat model, тесты, runbooks и настройки CI.

**Нормативная цель:** любой PR и любой релиз должны иметь проверяемые свидетельства (evidence), что (a) угрозы осмыслены и зафиксированы, (b) контрольные меры реализованы и проверены, (c) инцидент можно обнаружить, локализовать и разобрать по логам, (d) уязвимости обрабатываются по правилам. Это соответствует SSDF (PW.1/7/8/9 и RV.*) и OSPS Baseline для политик SAST/SCA и работы с уязвимостями. citeturn7view2turn7view3turn9view0turn5view1

### Этап Design / Change planning

**Обязательные артефакты (MUST при создании сервиса, а также при изменениях, влияющих на trust boundaries/данные/доступы):**
1) **Threat model** как living-doc. Минимальный формат:
- DFD (можно “diagram-as-code”), где явно показаны внешние сущности, процессы, хранилища, потоки данных и trust boundaries. citeturn20view0turn1search5  
- Таблица: Assets → Threats (например, STRIDE-подход как техника генерации угроз упоминается в OWASP cheat sheet) → Mitigations → Verification (какой тест/чек/лог подтверждает). citeturn20view0turn7view2  
- Раздел “Review & validation”: что проверено, какие риски приняты и почему (risk acceptance должен быть явным). citeturn20view0turn17view0  

2) **Security requirements** в виде “контрольного списка” для сервиса: для web/API-контролей хорошей базой является OWASP ASVS (можно выбирать subset под уровень риска). ASVS прямо позиционируется как список требований для secure development и база для verification. citeturn12search8turn12search4turn12search0

3) **Abuse cases (опционально, SHOULD для high-risk фич)**: формулируйте 3–10 “наиболее вероятных/потенциально дорогих” misuse-сценариев на фичу и связывайте их с тестами и логированием. При этом важно помнить, что OWASP помечает abuse-case cheat sheet как historical и отмечает, что подход может быть heavyweight — поэтому default должен быть lightweight. citeturn20view1

**Automation hook (SHOULD):** шаблон PR должен содержать чек “обновлён threat model / security requirements / abuse cases”, а codeowners должны назначать security reviewer’а на изменения в чувствительных областях (authn/authz, crypto, input parsing, data export). Обоснование: SSDF рекомендует определять, когда нужен code review/analysis (PW.7.1) и вести triage/учёт найденных проблем (PW.7.2). citeturn7view3turn18view0

### Этап Pull Request / pre-merge gates

**Минимальный набор обязательных проверок до merge (MUST):**

1) **Code review + “security-aware review submission”**  
Код-ревью должно иметь контекст: ссылка на требования/дизайн/тестирование/threat model в описании PR. Это соответствует рекомендациям OWASP Code Review Guide о том, что в ревью-подаче должны быть ссылки на связанные документы (включая threat modeling) и тестирование, выполненное разработчиком. citeturn18view0  
Плюс: SSDF требует определить необходимость code review/analysis (PW.7.1) и выполнить их с triage найденных issues (PW.7.2). citeturn7view3

2) **SAST gate**  
Для template разумный default — CodeQL code scanning как автоматизированная проверка уязвимостей/ошибок. citeturn22view1  
Важно: OSPS Baseline требует иметь политику порогов remediation для SAST и блокировать изменения при нарушениях (кроме явно задокументированных suppressions как non-exploitable). citeturn5view1

3) **SCA / dependency vulnerabilities gate**  
Для Go: запуск govulncheck на PR как минимум на изменённых пакетах/модуле. Официальные материалы подчёркивают, что govulncheck “low-noise” и показывает, какие уязвимости реально достигаются из вашего кода. citeturn13view1turn11view0  
Для policy: OSPS Baseline требует политики порогов remediation для SCA findings и (для зрелых уровней) автоматической оценки всех изменений и блокировки при violations. citeturn5view1

4) **Secrets scanning gate**  
Минимум — platform push protection/secret scanning (если доступно), потому что push protection специально предназначен для блокировки попадания секретов в репозиторий. citeturn2search4turn2search17  
Дополнительно (особенно для переносимости CI) — запуск gitleaks на PR. citeturn15search0  
Это также согласуется с OSPS Baseline требованием предотвращать попадание незашифрованных секретов в VCS. citeturn4view0turn5view1

5) **Go security checks baseline**  
Официальная страница Go Security Best Practices рекомендует: регулярный скан уязвимостей (govulncheck), поддержание актуальной версии Go и зависимостей (с осторожностью), fuzzing, race detector и go vet. citeturn11view0  
Для template это превращается в обязательные CI jobs: `go test`, `go test -race`, `go vet ./...`, `govulncheck ./...`. citeturn11view0turn13view1

6) **Fuzzing smoke (обязателен для security-critical обработчиков)**  
Go-руководство по fuzzing описывает генерацию случайных данных для поиска crash/vuln-вызывающих входов и запуск через `go test -fuzz`. citeturn14view0  
SSDF в PW.8 прямо приводит fuzz testing как пример техники для проверки исполняемого кода. citeturn9view2turn9view0  
Практический gate: на PR — короткий fuzztime (например, секунды/десятки секунд) или прогон seed corpus, чтобы не убивать скорость CI; длинный прогон — по расписанию (nightly) или на release. citeturn14view0turn11view0

### Этап Release / pre-release gates

**Цель:** релиз должен быть “готов к инциденту”, т.е. и артефакты, и telemetry/логирование, и уязвимости — управляемы.

1) **Полный прогон security checks**  
SSDF рекомендует использовать автоматизированный toolchain для регулярного/непрерывного анализа и тестирования релизов (RV.1.2) и применять testing/review (PW.7/PW.8) как часть жизненного цикла. citeturn17view1turn9view0turn7view3

2) **Dependency update policy**  
Для GitHub-окружения: Dependabot поддерживает и security updates (PR для уязвимых зависимостей), и version updates (поддержание актуальности). citeturn2search1turn2search5  
При этом Go рекомендует держать зависимости актуальными, но подчёркивает, что обновления без review могут быть рискованными (баги/несовместимость/вредоносный код), поэтому политика должна требовать review+тесты на каждое обновление. citeturn11view0

3) **Vulnerability disclosure + triage процесс**  
SSDF требует иметь policy для disclosure и remediation (RV.1.3) и проводить risk-based оценку/приоритизацию/ремедиацию (RV.2). citeturn17view1turn17view0  
OSPS Baseline требует публиковать security contacts/process (security.md) и поддерживать приватный канал репортинга. citeturn5view1turn4view0  
Как “boring default” для template:  
- **SEV/CVSS-based triage**, где SLA измеряется временем до **triage** и временем до **mitigation/release**, но конкретные цифры — организационно-зависимы; SSDF задаёт именно принцип risk-based remediation, а не числа. citeturn17view0  
- Завести единый трекер security findings (issues/alerts) и требовать статуса (triaged/in progress/fixed/accepted). citeturn17view0turn2search26

4) **Incident readiness check как часть release**  
OWASP Logging Cheat Sheet рекомендует: документировать security logging механизмы в release документации, интегрировать мониторинг с incident response процессами и защищать логи от подделки/удаления. citeturn19view0  
CNCF подчёркивает критичность “actionable audit events” и немедленной пересылки логов в место, недоступное из компрометированных учёток (чтобы злоумышленник не “замёл следы”). citeturn16view0

### Этап Runtime / incident readiness (подготовка до инцидента)

**Audit log vs security log (что должно быть в template):**
- OWASP разделяет security event logging и другие типы логов/трейлов (process monitoring, audit trails), указывая, что они часто имеют разные цели и их имеет смысл держать раздельно. citeturn19view0  
- OWASP Top 10 (A09) даёт конкретные failure-моды: не логируются auditable events, логи хранятся только локально, нет мониторинга и эскалации, нет incident response плана. Это можно напрямую превратить в “минимальные требования к логированию/мониторингу”. citeturn21view0  

**Минимальный набор audit/security events (MUST в template):**
- AuthN/AuthZ события: успешные/неуспешные логины, отказы по доступу, изменения прав/ролей. citeturn21view0turn19view1  
- Изменения данных с бизнес-ценностью: создание/изменение/удаление “sensitive objects”, экспорты данных. citeturn19view0turn19view1turn21view0  
- Административные операции: изменение конфигурации безопасности, включение/выключение механизмов мониторинга/аудита, действия с секретами (без записи самих секретов). citeturn19view0turn21view0turn16view0  
- Каждое событие должно быть коррелируемым (request_id/trace_id), иметь контекст пользователя/сервиса и быть в формате, пригодном для log management. citeturn21view0turn19view0turn16view0  

**Log management и защита логов (SHOULD):**
- Руководство NIST по log management подчёркивает необходимость “sound log management” и практические подходы к планированию/внедрению/поддержке управления логами. citeturn8search1turn8search5  
- OWASP Logging Cheat Sheet: защита логов от tampering/удаления, обнаружение остановки логирования, интеграция в централизованные системы и немедленное alerting по серьёзным событиям. citeturn19view0  

**Incident response программа (MUST на уровне организации сервиса/команды):**
- NIST 800-61r3 позиционирует IR как часть cybersecurity risk management и предлагает high-level lifecycle через функции CSF 2.0, подчёркивая непрерывное улучшение и необходимость политик/процессов/ролей. citeturn10view0turn6view1  
- Документ также отмечает ценность exercises/tests (в т.ч. tabletop discussions) и ссылается на NIST 800-84 как на руководство по TT&E. citeturn10view2turn8search4  

Практический минимум для template repo:
- **Runbook** “Security incident — service-level”: как объявить инцидент, где смотреть логи/метрики, как отключить опасный путь (feature flag / denylist / rate limit), как валидировать компрометацию, как собирать доказательства. (Нормативность опирается на требования NIST к интеграции IR и документированию/after-action reporting.) citeturn10view0turn10view1  
- **Tabletop exercise** (например, раз в квартал или при существенных изменениях модели угроз): сценарии должны быть привязаны к актуальным abuse cases/threat model. TT&E руководство описывает проектирование и оценку таких упражнений. citeturn8search4turn8search0  

## Набор правил MUST / SHOULD / NEVER для LLM

Эти правила предназначены для прямого копирования в “LLM-instructions” внутри репозитория (например, `docs/llm/security.md`). Они формулируют, что модель **обязана** произвести помимо кода.

### MUST

- **Всегда начинать изменения с извлечения security-контекста**: активы, доверенные/недоверенные границы, входы/выходы, зависимости, операции с данными. Threat modeling должен быть ранним, повторяемым и встроенным в SDLC. citeturn20view0turn7view2  
- **Обновлять threat model** при любом изменении trust boundary, нового внешнего интеграционного канала, нового типа данных, новой роли/прав, новой critical операции. citeturn20view0turn5view2  
- **Добавлять verification evidence**, а не только “реализацию”: security-тесты, негативные кейсы, fuzz tests для парсеров/валидаторов/кодеков, и/или правила CodeQL/SAST, которые ловят класс ошибок. citeturn9view0turn14view0turn7view3  
- **Всегда включать code review/analysis по правилам репозитория**, и фиксировать найденные проблемы в трекере (включая риск и план remediation). citeturn7view3turn17view0  
- **Для Go-проектов всегда запускать govulncheck** (или эквивалентный шаг CI) и при наличии findings добавлять triage-комментарий в PR (affected/unaffected, путь вызова, план). citeturn11view0turn13view1turn0search7  
- **Генерировать audit/security события** для операций, влияющих на безопасность, и включать контекст для расследований (кто/что/когда/результат/correlation id), не включая секреты. Недостаточное логирование — известный класс проблем для обнаружения breach. citeturn21view0turn19view0turn16view0  
- **Считать “готовность к инциденту” частью Definition of Done**: изменения, влияющие на detection/response, должны сопровождаться обновлением runbook или инструкций. citeturn10view0turn19view0  
- **Поддерживать security contacts и приватный канал репортинга уязвимостей** (как минимум — SECURITY.md), если сервис/репозиторий живёт дольше одного спринта. citeturn5view1turn17view1  

### SHOULD

- **Использовать OWASP logging vocabulary/конвенции событий** для унификации мониторинга и алёртинга между сервисами. citeturn19view1  
- **Делать “двухскоростной” security testing**: быстрые проверки на PR, расширенные (длинный fuzz, полный скан артефактов) — по расписанию/на релиз, чтобы не убивать скорость delivery. citeturn14view0turn9view0turn5view1  
- **Трактовать обновления зависимостей как изменения безопасности**: каждый bump должен проходить review+тесты; обновляться регулярно, но осторожно (Go прямо предупреждает о рисках “обновиться без ревью”). citeturn11view0turn2search1  
- **Автоматизировать “repo hygiene”**: branch protection, required status checks, отсутствие прямых коммитов в primary branch, чтобы enforcement был техническим, а не “на словах”. citeturn4view0turn22view0  
- **Писать abuse cases только там, где они дают проверяемые тесты**, и держать их небольшими и привязанными к риск-областям (OWASP отмечает риск “heavyweight”). citeturn20view1  

### NEVER

- **Никогда не размещать секреты/ключи/токены в коде, тестах, примерах конфигов** (даже “заглушки, похожие на секреты” без allowlist), потому что это ломает secret scanning и повышает риск утечки. citeturn2search4turn4view0turn15search0  
- **Никогда не предлагать обход security gates “ради удобства”** (например, отключить secret scanning/SAST, “задушить” алёртинг, хранить логи только локально). Это прямо соответствует failure-модам из OWASP A09. citeturn21view0turn5view1  
- **Никогда не оставлять findings без triage**: если сканер показал issue, нужно либо исправить, либо задокументировать non-exploitable/accepted risk с обоснованием и ссылкой на threat model/mitigation. Risk-based remediation — часть SSDF. citeturn17view0turn5view1  
- **Никогда не генерировать “security requirements” вида “сервис должен быть безопасным” без конкретизации** — OWASP прямо указывает, что такие требования бесполезны; вместо этого нужно перечислить атаки/контрмеры/проверки. citeturn20view1turn12search8  

## Concrete good / bad examples и типовые anti-patterns LLM

Ниже примеры, которые можно прямо включать в template как “reference patterns” для LLM.

### Пример fuzzing для критичного парсинга

**Good (fuzz test как “verification artifact”, seed corpus + property checks):**
```go
package transport

import (
	"bytes"
	"encoding/json"
	"testing"
)

type Payload struct {
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

func DecodePayload(b []byte) (Payload, error) {
	var p Payload
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Payload{}, err
	}
	// пример простейших инвариантов
	if p.UserID == "" || p.Action == "" {
		return Payload{}, errInvalidPayload
	}
	return p, nil
}

func FuzzDecodePayload(f *testing.F) {
	f.Add([]byte(`{"user_id":"u1","action":"login"}`))
	f.Add([]byte(`{"user_id":"","action":"x"}`))
	f.Fuzz(func(t *testing.T, in []byte) {
		_, _ = DecodePayload(in)
		// цель: отсутствие паник, отсутствие зависаний, устойчивость к мусорным входам
	})
}
```

Почему это “good”: официальный tutorial по fuzzing описывает `FuzzXxx`, seed corpus (`f.Add`) и запуск через `go test -fuzz`, а SSDF приводит fuzz testing как пример security testing исполняемого кода. citeturn14view0turn9view2turn9view0

**Bad (анти-паттерн):** “у нас есть unit tests, значит fuzz не нужен” для кода, который парсит недоверенный ввод. В Go security best practices fuzzing прямо рекомендован как способ находить edge-case exploits. citeturn11view0turn14view0

### Пример audit/security logging с разделением целей

**Good (структурированные события + пригодность для мониторинга/IR):**
```go
type AuditEvent struct {
	Time      string            `json:"time"`       // ISO8601 with offset
	Event     string            `json:"event"`      // e.g. "authn_login_success"
	ActorID   string            `json:"actor_id"`   // user/service principal
	Target    string            `json:"target"`     // resource
	RequestID string            `json:"request_id"` // correlation
	Result    string            `json:"result"`     // success/fail
	Meta      map[string]string `json:"meta,omitempty"`
}
```

Почему это “good”: OWASP Logging Cheat Sheet требует security-oriented application logging, совместимости с log management и интеграции мониторинга с incident response; OWASP Top 10 (A09) перечисляет auditable events как обязательные; CNCF подчёркивает важность “actionable audit events” для decision trees/incident response. citeturn19view0turn21view0turn16view0

**Bad (анти-паттерн):** “логируем только локально” или “логируем всё подряд, включая чувствительные данные”. A09 прямо указывает, что локальное хранение и недостаточная/неправильная запись событий ломают обнаружение/эскалацию; Logging Cheat Sheet подчёркивает необходимость выбирать, что логировать, и защищать логи. citeturn21view0turn19view0

### Типичные LLM-ошибки, которые должны ловиться процессом

- **LLM генерирует “обходы” security controls** (“просто отключим проверку/скан, чтобы прошло CI”). Это должно блокироваться политикой SAST/SCA и required checks; OSPS Baseline задаёт именно такой подход: violations блокируют merge/release (кроме задокументированных suppressions). citeturn5view1  
- **LLM добавляет новый endpoint/интеграцию без обновления threat model и security tests.** OWASP рекомендует threat modeling как поддерживаемый living-doc, а OSPS Baseline прямо требует threat modeling/attack surface analysis к релизу; SSDF включает risk modeling (PW.1.1). citeturn20view0turn5view2turn7view2  
- **LLM “забывает” про incident readiness** (нет audit events, нет корреляции, нет ссылок на runbook). A09 и CNCF подчёркивают, что без правильных audit/security events и процессов эскалации IR фактически неработоспособен. citeturn21view0turn16view0turn10view0  

## Review checklist для PR/code review и что оформить отдельными файлами в template repo

### Review checklist

Этот чеклист — компромисс между “коротко” и “покрывает основные failure-моды”. Он должен быть встроен в PR template и/или code review guide.

**Security design / threat model**
- Изменение затрагивает trust boundary / ввод / права / данные? Тогда threat model обновлён и содержит mitigations + verification. citeturn20view0turn5view2  
- Для high-risk изменений есть хотя бы 1–3 abuse-like сценария с негативными тестами (если применимо). citeturn20view1  

**Verification gates (до merge)**
- Есть evidence code review/analysis и triage найденных проблем (в issues/alerts). citeturn7view3turn17view0  
- SAST и SCA policy соблюдены (нет “необъяснённых” suppressions). citeturn5view1turn22view1  
- Govulncheck пройден/triaged, findings не оставлены “как есть”. citeturn11view0turn13view1turn17view0  
- Secrets scanning не сработал; при инциденте с секретом — сначала ротация, затем cleanup, фикс в CI/правилах. (Про предотвращение утечек — platform push protection и политика OSPS.) citeturn2search4turn4view0turn15search0  
- Для парсеров/валидаторов есть fuzz smoke или хотя бы seed corpus прогон. citeturn14view0turn9view2  

**Incident readiness**
- Добавлены/обновлены security/audit события для критичных действий; события не содержат секретов и пригодны для корреляции. citeturn21view0turn19view0turn16view0  
- Обновлён runbook/операционная заметка, если менялись сигналы/алёрты/пути расследования. citeturn19view0turn10view0  

### Что оформить отдельными файлами в template repo

Ниже — список “готовых doc/config артефактов”, которые делают процесс LLM-совместимым (минимум догадок, максимум явных контрактов).

**Документы (docs/…)**
- `docs/security/process.md` — этот lifecycle: обязательные этапы, gate’ы до merge/release, как triage’ить findings, что считается evidence. (SSDF PW.7/PW.8/RV.* как опорный каркас.) citeturn7view3turn9view0turn17view0  
- `docs/security/threat-modeling.md` — как делать DFD, trust boundaries, STRIDE-проход, шаблон таблицы Threat→Mitigation→Verification. citeturn20view0turn1search5turn7view2  
- `docs/security/logging-and-audit.md` — что логировать, что исключать, формат событий, требования к централизованной доставке и защите логов. citeturn19view0turn21view0turn16view0turn8search1  
- `docs/security/incident-response.md` — service-level IR runbook + как маппится на орг-процесс, шаблон after-action, где хранить артефакты. (Опереться на NIST 800-61r3.) citeturn10view0turn10view1  
- `docs/security/tabletop-exercises.md` — сценарии упражнений, частота, шаблон проведения/оценки (NIST 800-84). citeturn8search4turn8search0turn10view2  
- `docs/llm/security.md` — MUST/SHOULD/NEVER для модели, плюс “Definition of Done: security edition”. citeturn7view3turn20view0turn21view0  

**Политики в корне репозитория**
- `SECURITY.md` — security contacts, приватный канал для репортинга, ожидания по disclosure/triage (OSPS-VM-02/03 и SSDF RV.1.3). citeturn5view1turn17view1  

**CI/CD (пример под GitHub, но структура переносима)**
- `.github/workflows/codeql.yml` — CodeQL scanning. citeturn22view1turn2search6  
- `.github/workflows/govulncheck.yml` — govulncheck job. citeturn11view0turn13view1  
- `.github/workflows/security-baseline.yml` — секреты (gitleaks), линтеры, `go test -race`, `go vet`, fuzz smoke. citeturn11view0turn15search0turn14view0  
- `.github/dependabot.yml` — dependency updates policy. citeturn2search1turn2search11  

**Repo conventions**
- PR template с обязательными ссылками на threat model / security requirements / tests (OWASP Code Review Guide) и чеклистом security gates. citeturn18view0turn7view3  
- Настройки branch protection / required checks и периодический прогон OpenSSF Scorecard как “индикатора” hygiene. citeturn4view0turn22view0