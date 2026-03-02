# Clean code и readability в Go для production-ready template микросервиса и LLM-инструкций

## Scope

Этот стандарт применим, когда вы делаете **greenfield template** для микросервиса на Go, который будут развивать люди (включая новичков в Go) и активно дополнять с помощью LLM‑инструментов; ключевая цель — чтобы изменения были **предсказуемо читаемыми**, легко ревьюились и естественно вписывались в экосистему Go (gofmt/godoc/go test), без «религиозных» обсуждений стиля и без скрытой магии. citeturn8search1turn2search4turn3search12

Этот стандарт **не** стоит применять «строго как закон» в следующих случаях:

- **Сгенерированный код** (protobuf, OpenAPI, mocks, sqlc и т.п.): такие файлы часто нарушают правила именования/комментариев и должны рассматриваться отдельно; например, Go Code Review Comments прямо оговаривает исключения для кода, сгенерированного protobuf‑компилятором, в части правил про initialisms. citeturn9view0turn10view0  
- **Крайне низкоуровневые/перформанс‑критичные участки**, где сознательно выбирают менее «приятную» форму ради измеряемой выгоды; при этом даже там предпочтительно сохранять gofmt‑совместимую форму и семантически ясные имена, а оптимизации — объяснять комментариями/бенчмарками (иначе ревью и сопровождение становятся дороже). citeturn2search1turn9view0turn3search12  
- **Контракты внешних API/SDK**, которые диктуют форму публичных имен/типов (например, строго по протоколу), — тогда отклонения фиксируются локальными исключениями и документацией пакета/типа. citeturn4view2turn9view0  

## Recommended defaults для greenfield template

Ниже — «boring, battle‑tested defaults» именно для **чистоты кода и читаемости** в Go (то, что удобно почти напрямую положить в `docs/` и в LLM‑instruction).

**Форматирование и “consistent code shape”**
- Весь код **форматируется gofmt**; размер отступов/выравнивание не настраиваются «под команду»: `gofmt` использует табы для отступов и пробелы для выравнивания, и именно этот стиль ожидаем во всех PR. citeturn2search1turn8search2turn2search4  
- Для шаблона репозитория по умолчанию лучше включать **goimports** вместо чистого gofmt: это надмножество gofmt, которое ещё и приводит `import`‑блоки к канонической форме (добавляет/удаляет импорты, группирует, кладёт стандартную библиотеку первой группой). citeturn8search0turn8search1turn10view0  

**Именование “по‑Go‑шному”**
- **Package names**: короткие, нижний регистр, одно слово, без `under_scores` и `mixedCaps`; имя пакета становится префиксом для всего экспортируемого, поэтому не надо «дублировать» смысл пакета в именах типов и функций (анти‑статтеринг). citeturn5view0turn4view4turn9view0  
- **Don’t steal good names from the user**: не называйте пакет так, что он будет постоянно конфликтовать с хорошими локальными именами (классический пример из Go Blog: `bufio`, а не `buf`). citeturn4view4  
- **MixedCaps** для многословных идентификаторов вместо `snake_case`; это распространяется и на неэкспортируемые константы/переменные. citeturn5view0turn9view0turn10view0  
- **Initialisms**: аббревиатуры имеют единый регистр (`ServeHTTP`, `appID`, не `ServeHttp`/`appId`). citeturn9view0turn10view0  
- **Getters**: если есть поле `owner`, геттер называется `Owner()`, а не `GetOwner()`; `SetOwner()` допустим для сеттера. citeturn5view0  
- **Канонические имена/сигнатуры**: не придумывайте «java‑style» (`ToString`) там, где у Go есть стандартный смысл (`String() string`). citeturn5view0  
- **Receiver names**: короткие (часто 1–2 буквы), без `this/self/me`, единообразные между методами типа. citeturn9view0  
- **Local variable names**: короткие в малом скоупе; чем дальше место использования от объявления, тем более описательным должно быть имя (правило «дистанции»). citeturn9view0  

**Комментарии, пакетная документация, публичная эргономика API**
- **Doc comments** обязательны для экспортируемых символов; doc comment — это комментарий непосредственно перед декларацией (без пустых строк), и именно они составляют документацию пакета/символа в инструментах `go doc`/pkg.go.dev. citeturn7view3turn5view2turn4view2  
- Комментарии к декларациям пишутся **полными предложениями**, начинаются с имени описываемого объекта и заканчиваются точкой — так они корректно выглядят в godoc. citeturn10view0turn5view2  
- **Package comment** должен быть рядом с `package ...` без пустой строки; по соглашению первая фраза начинается со слова `Package ...`. Для многофайлового пакета пакетный комментарий лучше держать в одном файле (типично `doc.go`), иначе комментарии конкатенируются. citeturn9view0turn4view2turn3search0  
- Документация типов должна явно фиксировать: что означает «экземпляр типа», гарантии по конкурентному доступу (если они есть), и смысл нулевого значения, если он неочевиден. citeturn7view2turn4view2  

**Ошибки и читаемость error flow**
- Ошибки — это значения: их нужно возвращать/обрабатывать, а не «прятать»; игнорировать error через `_` нельзя. citeturn4view3turn10view0turn5view3  
- `panic` не используется для обычной обработки ошибок в прикладном коде (исключения редки и должны быть мотивированы). citeturn10view0turn5view3turn9view0  
- **Ветвление по ошибкам** оформляется так, чтобы «нормальный путь» был с минимальным уровнем вложенности: сначала early return на ошибке, потом основной код. citeturn9view0turn10view0  
- **Error strings**: без заглавной буквы (если это не собственное имя/акроним) и без пунктуации в конце — потому что ошибки часто печатаются внутри другого контекста. citeturn10view0turn4view1  
- При возможности error string должен обозначать происхождение/операцию (например, `image: unknown format`), чтобы сообщение было информативным «далеко от места возникновения». citeturn5view3  
- Для современного idiomatic Go в template разумно по умолчанию использовать **цепочки ошибок**: `%w` для wrapping + `errors.Is`/`errors.As` для проверок. citeturn1search2turn3search15turn5view3  

**nil‑семантика и «полезное нулевое значение»**
- При проектировании типов и структур данных следует стремиться к тому, чтобы **нулевое значение было готово к использованию** (как у `bytes.Buffer` или `sync.Mutex` в примере Effective Go); если нулевое значение имеет полезный смысл, но он неочевиден — его надо документировать. citeturn7view0turn7view2  
- Пустой slice по умолчанию объявляется как `var s []T` (nil slice), а не `s := []T{}`; при этом публичные API не должны различать nil slice и пустой slice на уровне смыслов/контрактов. citeturn10view0turn4view1  
- Исключение: когда slice сериализуется в JSON, **nil slice кодируется как `null`**; если контракт API требует `[]`, то нужно обеспечивать non‑nil пустой slice. citeturn12search0turn10view0turn4view1  
- Для map важно помнить: **nil map эквивалентна пустой при чтении**, но в неё нельзя добавлять элементы; это влияет на проектирование zero‑value и на читаемость (меньше «Init()» вокруг). citeturn14view3turn7view0  

**Тесты как часть читаемости**
- По умолчанию тесты пишутся так, чтобы при падении они давали максимально диагностичное сообщение: что было входом, что получили, что ожидали; порядок «got vs want» должен быть консистентным. citeturn9view0turn10view0  
- Table‑driven tests — базовая идиома Go для сокращения дублирования и повышения читаемости тестов. citeturn0search3turn9view0  

## Decision matrix и trade-offs

| Решение | Bor ing default для template | Когда выбрать иначе | Ключевой trade-off / риск |
|---|---|---|---|
| nil slice vs empty slice в публичном API | Считать эквивалентными, наружу не «протаскивать» различие; хранить внутри как nil по умолчанию | Если контракт JSON/клиентов требует `[]` (не `null`) | Скрытая несовместимость в JSON: nil slice → `null`. citeturn10view0turn12search0turn4view1 |
| Ошибки: sentinel vs typed vs wrapping | Wrapping через `%w` + проверки `errors.Is/As`; sentinel — для устойчивых «классов» ошибок; typed — когда нужно вытянуть детали | Если ошибка — часть протокола/контракта и требуется стабильная структура (type) | Слишком «плоские» строки ухудшают диагностику; слишком сложная иерархия типов усложняет API. citeturn1search2turn5view3turn10view0turn3search15 |
| Именование геттеров | Без `Get` (например, `Owner()`) | Если вынуждает внешний интерфейс/генератор | “Get” выглядит неидиоматично, ухудшает читабельность вызовов. citeturn5view0 |
| Интерфейсы для тестируемости | Интерфейсы определяются в пакете‑потребителе; продьюсер возвращает конкретный тип | Плагины/инверсия зависимостей действительно требуют интерфейса как части контракта | Раннее/избыточное абстрагирование делает код менее ясным и ломает расширяемость (нужно менять интерфейс). citeturn9view0turn10view0 |
| Named returns и naked return | По умолчанию избегать; использовать только когда имена реально улучшают godoc/ясность или нужны для `defer` | Короткие функции‑утилиты, либо случаи с `defer`, меняющим возвращаемые значения | Naked return в «средних» функциях ухудшает читаемость; именованные результаты делают публичный API более «шумным». citeturn9view0turn4view0 |
| Комментарии vs “self-documenting code” | Комментарии обязательны для экспортируемого; внутри — только там, где намерение неочевидно из кода | Алгоритмически сложные места, неочевидные инварианты, требуются ссылки на спецификации | Недокомментирование ломает godoc; перекомментирование создаёт расхождение комментариев и кода. citeturn7view3turn10view0turn5view2 |
| Импорты: alias/dot/blank | goimports, минимум alias; `import .` — практически никогда; blank import — только для side effects в `main` или специальных тестах | Исключения: коллизия имён; тест‑пакет с циклическими зависимостями | `import .` делает код трудно читаемым (неясно, откуда символ); blank import размывает причинность кода. citeturn10view0turn5view0 |
| “Utility” пакеты и нейминг | Запрещать `util/common/misc/...`; называть по домену/назначению | Узкий технический пакет с чёткой областью (например, `httputil` в stdlib — исторический пример, но в новых пакетах лучше избегать) | “util” превращается в свалку и ухудшает навигацию/понимание. citeturn9view0turn4view4 |

## Набор правил MUST / SHOULD / NEVER для LLM

Ниже — формулировки в стиле «инженерного стандарта» для LLM‑instruction. Они специально заточены под генерацию кода **без лишних догадок**: модель должна выбирать канонические формы Go, которые ожидает ревьюер и tooling.

**MUST**
- MUST запускать/соблюдать `gofmt` (или `goimports`, если он принят как formatter) для всего Go‑кода. citeturn8search1turn2search1turn8search0  
- MUST использовать `MixedCaps/mixedCaps` для многословных идентификаторов и соблюдать правила initialisms (`HTTP`, `URL`, `ID`). citeturn5view0turn9view0turn10view0  
- MUST выбирать package names: нижний регистр, коротко, без `_` и без `mixedCaps`; MUST избегать бессмысленных имён пакетов вроде `util/common/misc/...`. citeturn5view0turn4view4turn9view0  
- MUST избегать «статтеринга»: экспортируемые имена должны учитывать, что вызываются как `pkg.Name` (например, `bufio.Reader`, а не `BufReader`). citeturn5view0turn9view0  
- MUST писать doc comments для каждого экспортируемого имени; doc comment должен быть непосредственно перед декларацией без пустых строк. citeturn7view3turn10view0turn9view0  
- MUST писать комментарии к декларациям полными предложениями, начиная с имени объекта и заканчивая точкой. citeturn10view0turn5view2  
- MUST обеспечивать package comment (обычно в `doc.go`), расположенный рядом с `package` без пустых строк; первая фраза package comment должна начинаться с `Package ...`. citeturn9view0turn4view2turn3search0  
- MUST всегда проверять возвращаемые `error`; нельзя «выбрасывать» ошибки через `_`. citeturn10view0turn4view3  
- MUST оформлять error flow так, чтобы нормальный путь имел минимальную вложенность (early return на ошибках). citeturn9view0turn10view0  
- MUST формировать error strings без заглавной буквы и без пунктуации на конце (если это не proper noun/акроним); ошибки должны хорошо вставляться в внешний контекст. citeturn10view0turn4view1  
- MUST, добавляя контекст к ошибке, использовать wrapping и сохранять причину (например, `%w`), чтобы дальше можно было применять `errors.Is/As`, когда это уместно. citeturn1search2turn5view3  
- MUST проектировать типы так, чтобы нулевое значение имело полезный смысл (или документировать его смысл). citeturn7view0turn7view2  
- MUST помнить семантику nil: nil slice/empty slice обычно эквивалентны по `len/cap`, но при JSON‑маршалинге nil slice → `null`. citeturn10view0turn12search0turn4view1  
- MUST учитывать, что nil map нельзя модифицировать (добавление элементов требует `make`). citeturn14view3  
- MUST писать тесты с диагностичными сообщениями (вход → got → want); по умолчанию использовать table‑driven подход там, где много кейсов. citeturn9view0turn0search3  

**SHOULD**
- SHOULD предпочитать `goimports` (как formatter) для поддержки канонической структуры `import`‑блоков. citeturn8search1turn8search0turn10view0  
- SHOULD избегать alias‑импортов, кроме случаев коллизий; стандартная библиотека — первая группа импортов. citeturn10view0turn4view4  
- SHOULD называть геттеры без `Get` (например, `Owner()`), а сеттеры — `SetX()`, если они действительно нужны. citeturn5view0  
- SHOULD использовать «канонические» имена методов и сигнатуры (`String`, `Read`, `Write`…), не вводя альтернативы вроде `ToString`. citeturn5view0  
- SHOULD держать имена локальных переменных короткими в малом скоупе; чем шире область видимости/дальше использование — тем описательнее имя. citeturn9view0  
- SHOULD использовать короткие и единообразные имена ресивера, не превращая его в «особый» объект (`this/self`). citeturn9view0turn7view2  
- SHOULD избегать именованных результатов и naked returns в «средних» и больших функциях; если имена результатов нужны, они должны добавлять ясность в godoc/контракт. citeturn9view0turn4view0  
- SHOULD избегать “in-band errors” (магических `-1`, `""`, `nil` как «ошибка») в публичных API; возвращать дополнительное значение (`error` или `ok bool`) последним результатом. citeturn10view0  
- SHOULD по умолчанию проектировать интерфейсы со стороны потребителя; не объявлять интерфейсы «на стороне реализации» только ради моков. citeturn9view0turn10view0  
- SHOULD предпочитать nil slice объявлению `[]T{}` в коде домена/внутренней логике (если нет требований протокола), и не заставлять вызывающего различать эти формы. citeturn10view0turn4view1  

**NEVER**
- NEVER использовать `panic` для обычной обработки ошибок в прикладных путях выполнения; `panic` не должен быть частью контракта функции/пакета. citeturn10view0turn5view3turn9view0  
- NEVER игнорировать `error` через `_` «потому что, кажется, тут неважно». citeturn10view0turn4view3  
- NEVER писать error strings с заглавной буквы и точкой в конце (кроме исключений с proper nouns/акронимами). citeturn10view0turn4view1  
- NEVER использовать `import .` в production‑коде; допустимое исключение — редкие тестовые файлы вне пакета при циклических зависимостях. citeturn10view0turn5view0  
- NEVER вводить `snake_case` для Go‑идентификаторов и имён пакетов. citeturn5view0turn4view4turn9view0  
- NEVER называть пакеты `util/common/misc/types/interfaces/api` без предметного смысла. citeturn9view0  
- NEVER делать «интерфейсы для моков» на стороне реализатора, возвращая интерфейс вместо конкретного типа без причины. citeturn9view0turn10view0  
- NEVER передавать указатели «чтобы сэкономить байты» для фиксированных по размеру значений (например, `*string`, `*io.Reader`) без причины — это ухудшает читаемость API и часто не даёт выигрыша. citeturn9view0  

## Concrete good / bad examples, где уместно — на Go

**Пакеты и статтеринг**

Плохо: пакет с подчёркиваниями + типы дублируют имя пакета и роль.

```go
package user_service

type UserServiceClient struct {
    // ...
}

func NewUserServiceClient() *UserServiceClient { /* ... */ return nil }
```

Хорошо: короткое имя пакета; экспортируемые имена учитывают префикс `users.`; `New` уместно, когда в пакете доминирует один основной тип/конструктор.

```go
package users

type Client struct {
    // ...
}

func New() *Client { /* ... */ return &Client{} }
```

Пакеты должны быть lower case без `under_scores/mixedCaps`, а экспортируемые имена не должны повторять имя пакета; это прямо следует из Effective Go и Go Blog про package names. citeturn5view0turn4view4turn9view0  

**Геттеры без `Get`**

Плохо:

```go
type Object struct {
    owner string
}

func (o *Object) GetOwner() string { return o.owner }
```

Хорошо:

```go
type Object struct {
    owner string
}

func (o *Object) Owner() string { return o.owner }
```

`Get` в названии геттера неидиоматичен в Go — правило из Effective Go. citeturn5view0  

**Канонические имена методов и интерфейсов**

Плохо: «перенос» соглашений из других языков.

```go
type IReader interface {
    ReadBytes() ([]byte, error)
}

func (x Thing) ToString() string { return "..." }
```

Хорошо: интерфейсы‑агентные существительные и стандартные имена (`Reader`, `String`).

```go
type Reader interface {
    Read(p []byte) (int, error)
}

func (x Thing) String() string { return "..." }
```

Effective Go фиксирует суффикс `-er` для одно‑методных интерфейсов и призывает не изобретать имена вроде `ToString`, когда есть каноническое `String`. citeturn5view0  

**Error strings, контекст и wrapping**

Плохо: ошибка как «предложение», плюс теряется причина, плюс нет возможности корректно различать классы ошибок.

```go
func Load(path string) error {
    b, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("Failed to read file.")
    }
    _ = b
    return nil
}
```

Хорошо: сообщение без заглавной буквы и точки, добавляется контекст операции, причина сохраняется через wrapping.

```go
func Load(path string) error {
    b, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("read %s: %w", path, err)
    }
    _ = b
    return nil
}
```

Почему так: стиль error string (lowercase/no punctuation) задаётся CodeReviewComments; требование, чтобы error string был информативным «далеко от места возникновения» — в Effective Go; wrapping и последующая проверка через `errors.Is/As` — официальный подход начиная с Go 1.13. citeturn10view0turn5view3turn1search2  

**Indent error flow вместо вложенных `else`**

Плохо:

```go
if err != nil {
    // handle
} else {
    // normal path
}
```

Хорошо:

```go
if err != nil {
    // handle
    return err
}
// normal path
```

Это прямое правило из Go Code Review Comments: «держать нормальный путь с минимальной вложенностью». citeturn9view0turn10view0  

**nil slice, JSON и контракт API**

Плохо: возвращаем nil slice наружу, а JSON получает `null`, хотя контракт часто ожидает `[]`.

```go
type Resp struct {
    Items []string `json:"items"`
}

func Handler() Resp {
    var items []string // nil
    return Resp{Items: items}
}
```

Хорошо: если контракт требует массив, гарантируем non‑nil пустой slice.

```go
type Resp struct {
    Items []string `json:"items"`
}

func Handler() Resp {
    items := make([]string, 0) // encodes as []
    return Resp{Items: items}
}
```

`encoding/json` документирует: nil slice кодируется как `null`; CodeReviewComments отдельно отмечает это как частое исключение из «предпочитаем nil slice». citeturn12search0turn10view0turn4view1  

**“Make the zero value useful” на практике**

Плохо: тип требует обязательного `Init`, иначе паника/неочевидное поведение.

```go
type Cache struct {
    m map[string]string
}

func (c *Cache) Put(k, v string) {
    c.m[k] = v // panic if m is nil
}
```

Хорошо: нулевое значение работает; инициализация происходит лениво.

```go
type Cache struct {
    m map[string]string
}

func (c *Cache) Put(k, v string) {
    if c.m == nil {
        c.m = make(map[string]string)
    }
    c.m[k] = v
}
```

Effective Go рекомендует проектировать структуры так, чтобы zero value был полезен; спецификация напоминает, что nil map нельзя модифицировать — значит, либо делаем `make`, либо документируем, что zero value не готов. citeturn7view0turn14view3turn7view2  

**Doc comments, которые нормально выглядят в godoc**

Плохо: комментарий не начинается с имени объекта, не является полным предложением.

```go
// does stuff
func Process(x int) int { return x }
```

Хорошо:

```go
// Process returns x after applying the service's normalization rules.
func Process(x int) int { return x }
```

Правило «полные предложения, начать с имени, закончить точкой» — из CodeReviewComments; “doc comments как первичная документация” — из Effective Go и Go Doc Comments. citeturn10view0turn5view2turn7view3  

**Table-driven tests**

Плохо: много дублирования.

```go
func TestParseA(t *testing.T) { /* ... */ }
func TestParseB(t *testing.T) { /* ... */ }
```

Хорошо: таблица кейсов, цикл, и диагностичные сообщения.

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name string
        in   string
        want int
    }{
        {"empty", "", 0},
        {"one", "1", 1},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Parse(tt.in)
            if got != tt.want {
                t.Errorf("Parse(%q) = %d; want %d", tt.in, got, tt.want)
            }
        })
    }
}
```

Go Wiki описывает table‑driven tests как способ писать «чище»; CodeReviewComments отдельно подчёркивает требования к полезным сообщениям при падении тестов и прямо отсылает к table‑driven подходу. citeturn0search3turn9view0turn3search12  

**Что часто считается clean code в теории, но выглядит неидиоматично в Go**
- «Геттеры всегда `GetX()`» — в Go это специально не считается идиомой (достаточно `X()`), потому что экспортируемость уже различает поле и метод. citeturn5view0  
- «Интерфейсы объявлять в пакете реализации, чтобы мокать» — Go Code Review Comments прямо говорит делать наоборот: интерфейсы обычно принадлежат пакету‑потребителю, а продьюсер должен возвращать конкретные типы; иначе вы заранее фиксируете абстракцию без реального примера использования. citeturn9view0turn10view0  
- «Ошибки как исключения или паники» — `panic` не является нормальным механизмом управления ошибками в Go, и это ухудшает читабельность error flow. citeturn10view0turn5view3turn9view0  
- «Жёсткий лимит длины строки/функции» — в Go нет строгого лимита; переносы строк и границы функций должны диктоваться семантикой, а не счётчиком символов/строк. citeturn9view0turn8search2  

## Anti-patterns и типичные ошибки/hallucinations LLM

Ниже — ошибки, которые LLM чаще всего «галлюцинирует» из привычек других языков или из усреднённых “clean code” советов, и которые особенно токсичны для читаемости Go‑кода.

- **Не‑gofmt код** (нестандартные отступы, выравнивания, скобки “на новой строке”): Go‑экосистема предполагает gofmt как норму, а `gofmt` имеет чёткую специфику (табы/выравнивание пробелами). citeturn2search1turn2search4turn8search2  
- **Пакеты `util/common/...` как «свалка»**: ухудшает навигацию; CodeReviewComments прямо просит избегать таких имён. citeturn9view0  
- **`snake_case` и нарушения initialisms** (`appId`, `ServeHttp`): противоречит Effective Go и CodeReviewComments. citeturn5view0turn9view0turn10view0  
- **`GetOwner()` и “ToString()”**: неидиоматично; в Go есть канонические соглашения (`Owner()`, `String()`). citeturn5view0  
- **`panic` вместо возврата `error`** в обычной логике: ломает предсказуемость управления потоком и усложняет ревью. citeturn10view0turn5view3turn9view0  
- **Игнорирование ошибок через `_`**: запрещено CodeReviewComments; ухудшает не только надёжность, но и читаемость (скрывает контракт функции). citeturn10view0  
- **In-band errors** (возврат `""`, `-1`, `nil` «как ошибка» без отдельного `ok/error`): в Go обычно возвращают дополнительное значение, чтобы компилятор не дал «случайно» использовать невалидный результат. citeturn10view0  
- **Путаница nil interface vs typed nil**: LLM часто сравнивает `interface{}` с `nil` и делает неверные выводы. Спецификация: два interface значения равны, если совпадают динамический тип и значение, либо оба имеют значение `nil`; отсюда классическая ловушка «интерфейс не nil, хотя внутри typed nil». citeturn14view0turn14view1  
- **nil map используется как готовая коллекция**: чтение безопасно, но запись паникует; спецификация фиксирует, что в nil map нельзя добавлять элементы — значит, или `make`, или ленивый init. citeturn14view3turn7view0  
- **Случайная подмена nil slice/empty slice в JSON‑контрактах**: nil slice → `null`; многие API хотят `[]`, и это нужно обеспечивать явно. citeturn12search0turn10view0  
- **`import .` в production**: делает код трудно читаемым (неясно происхождение идентификатора); разрешённые кейсы — редкие тестовые ситуации. citeturn10view0turn5view0  
- **Использование `io/ioutil` как «привычной библиотеки»**: пакет официально помечен Deprecated начиная с Go 1.16; новый код должен использовать `io`/`os`. Это типичная LLM‑галлюцинация «из старых примеров». citeturn15search0turn15search3turn15search9  

## Review checklist для PR/code review

Этот чек‑лист можно использовать и человеком, и LLM‑ревьюером. Он ориентирован на **чистоту кода, читаемость и идиоматичность**.

- Проверить, что весь Go‑код отформатирован gofmt/goimports и импорт‑блоки каноничны (stdlib первой группой). citeturn8search1turn10view0turn2search1  
- Проверить, что package names в lower case, без `_`/`mixedCaps`, без “util/common…”, и что экспортируемые имена не «заикаются» относительно имени пакета. citeturn5view0turn4view4turn9view0  
- Проверить initialisms (`HTTP`, `URL`, `ID`) и MixedCaps во всех новых идентификаторах. citeturn9view0turn5view0turn10view0  
- Проверить, что новые экспортируемые символы имеют doc comments (без пустых строк перед декларацией), и что комментарии написаны полными предложениями, начинаются с имени объекта и заканчиваются точкой. citeturn7view3turn10view0turn5view2  
- Проверить наличие package comment рядом с `package` (обычно в `doc.go`), первая фраза начинается с `Package ...`; для multi‑file пакета — комментарий не размазан по файлам без причины. citeturn9view0turn4view2turn3search0  
- Проверить error flow: ошибки не игнорируются; `panic` не используется как обычная обработка ошибок; нормальный путь не утоплен во вложенные `else`. citeturn10view0turn9view0turn5view3  
- Проверить error strings: lower case / без точки; при добавлении контекста причина не теряется (wrapping), и остаётся возможность `errors.Is/As` там, где это важно. citeturn10view0turn1search2turn5view3  
- Проверить nil/zero‑value семантику: нулевое значение типов действительно «полезно» или документировано; нет записей в nil map; API не заставляет различать nil slice и empty slice; JSON‑контракты соблюдены (nil slice → `null` учтено). citeturn7view0turn14view3turn12search0turn10view0  
- Проверить тесты: сообщения об ошибках диагностичны (вход, got, want), таблицы кейсов используются там, где это уменьшает дублирование. citeturn9view0turn0search3  
- Проверить, что не добавили `import .` в production и blank import вне `main`/специальных тестов. citeturn10view0turn5view0  

## Что из результата нужно оформить отдельными файлами в template repo

Ниже — предлагаемая декомпозиция «как это положить в `docs/` и repo conventions», чтобы LLM могла ссылаться на конкретные файлы, а не “угадывать стиль”.

- `docs/standards/go-clean-code.md` — основной норматив: философия читаемости, решения по умолчанию, truth‑sources (Effective Go, CodeReviewComments, Go Doc Comments, spec). citeturn5view0turn4view1turn7view3turn14view0  
- `docs/standards/go-naming.md` — концентрат правил именования: package names, MixedCaps, initialisms, getters, receiver names, стеттеринг/anti‑stutter. citeturn4view4turn5view0turn9view0  
- `docs/standards/go-docs-and-comments.md` — правила doc comments, package comments, примеры корректного `doc.go`, типовые ошибки форматирования комментариев (включая правила, что doc comment — непосредственно перед декларацией). citeturn7view3turn4view2turn10view0turn9view0  
- `docs/standards/go-errors.md` — стиль ошибок: формат error strings, wrapping (`%w`), `errors.Is/As`, anti‑patterns (`panic`, игнор ошибок, in‑band errors), шаблоны сообщений. citeturn10view0turn1search2turn5view3turn4view3  
- `docs/standards/go-nil-and-zero-value.md` — практический гайд по nil/zero value: nil map, nil slice vs empty slice, JSON‑контракты, правила проектирования типов с полезным zero value. citeturn14view3turn10view0turn12search0turn7view0turn7view2  
- `docs/standards/go-testing-readability.md` — table‑driven tests, стиль сообщений, subtests, примеры «плохих» и «хороших» тестов. citeturn0search3turn9view0turn3search12  
- `docs/llm/GO_WRITING_RULES.md` — короткая “MUST/SHOULD/NEVER” версия (как в этом ответе), предназначенная для вставки в system‑prompt/репозиторный префикс. citeturn8search1turn10view0turn5view0turn7view0  
- `CONTRIBUTING.md` (или `docs/contributing.md`) — «как оформить PR»: требование gofmt/goimports, что будет проверяться, и ссылки на стандарты. citeturn2search4turn8search1  
- Автопроверки, чтобы стиль не был «на совести LLM»:  
  - `.editorconfig` (табы, LF, финальная новая строка), чтобы редакторы не спорили с gofmt. citeturn2search1turn8search2  
  - CI‑шаг `goimports -w` (или `gofmt -w`) + `go test ./...`, чтобы “consistent shape” и базовая корректность были автоматическими. citeturn8search0turn2search1turn3search12  
  - (Опционально) включить анализаторы/линтеры, которые принудительно ловят ключевые требования читаемости, например проверку package comment по правилам CodeReviewComments (в экосистеме она фигурирует как ST1000). citeturn0search5turn8search9turn12search1