# Mate YAML Language

This is a normative reference for the YAML syntax implemented by the current codebase. It documents only constructs accepted by the runtime today.

## Table of Contents

- [Logical Structure](#logical-structure)
  - [Model](#model)
  - [Coordinators](#coordinators)
  - [Producers](#producers)
  - [Share](#share)
  - [State](#state)
- [YAML Elements](#yaml-elements)
  - [`model`](#model-element)
  - [`coordinators`](#coordinators-element)
  - [`producers`](#producers-element)
  - [Activity element](#activity-element)
- [External Helpers](#external-helpers)
  - [Runtime Context](#runtime-context)
  - [Expression Evaluation](#expression-evaluation)
  - [State Helpers](#state-helpers)
  - [Share Helpers](#share-helpers)
  - [Config and Environment Access](#config-and-environment-access)
  - [HTML Template Helpers](#html-template-helpers)

## Logical Structure

### Model

All definitions loaded from one model source belong logically to one model. The physical YAML layout keeps `coordinators` and `producers` as top-level siblings of `model` so definitions can be composed from several files. `model` itself may occur only once.

```text
model
├── name
├── env
├── profiles
├── coordinators
│   ├── bootstrap
│   ├── schedules
│   └── routes
├── producers
│   └── producer
│       └── activities
│           └── activity
├── (share)
└── (state)
```

The logical and chapter order is coordinators, producers, share, then state. Share and state are logical resources without model-level YAML elements.

### Coordinators

Coordinators create activations for startup, schedules, and HTTP routes. Each coordinator selects an ordered list of named producers.

### Producers

Producers are named, reusable sequences of ordered activities. Branch and iterate activities may own nested activity lists.

### Share

Share is the model's in-memory key-value resource. It exists for the current model runtime and is cleared on restart. Activities use it through external helpers; activity-level `share` is inspection metadata, not a share declaration.

### State

State is the model's persistent key-value resource. It is stored in `model.db` in the model directory and survives runtime restarts. Activities use it through external helpers; activity-level `state` is inspection metadata, not a state declaration.

## YAML Elements

The accepted top-level elements are `model`, `coordinators`, and `producers`. Public collections are ordered lists whose entries carry an explicit `name`; lookup maps are internal and are not accepted as YAML syntax.

---

### `model` {#model-element}

```yaml
model:
  name: laddning
  env:
    API_URL:
      default: https://api.example.test
    API_TOKEN:
```

`model` configures the loaded model, its environment and an optional
stable URL namespace. `name`, when present, must contain lowercase letters or
digits separated by single hyphens and must be unique in the composed application.
A route `/test` for model `laddning` is registered at `/laddning/test`; static
files are below `/laddning/static/`. Dynamic administration operations are
ordinary model-defined routes; there is no `/_runtime/` HTTP API.

Persistence is intrinsic to the model directory. There is no model-level `state`,
`share`, `runtime_api`, database driver, DSN, or database-path property. Every
loaded model has equal access to the complete model-facing runtime API; runtime API
permissions are not configured in model YAML.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `name` | string | no | Lowercase URL slug used to namespace model routes and static files. |
| `env` | map | no | Environment declarations keyed by environment-variable name. |

---

#### `env`

```yaml
env:
  API_URL:
    default: https://api.example.test
  API_TOKEN:
```

Each key under `model.env` declares one environment variable. Environment names
must be non-empty. A null declaration is valid and means that no default is
supplied. Process environment values take precedence over defaults. Static
`env.NAME` references must name a declared variable.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `default` | string | no | Value used when the process environment does not contain the declared variable. |

---

#### Application profiles

```yaml
application:
  name: charge
  profiles:
    default: production
    available: [production, simulation]
```

`application.profiles` declares the application-wide profile names and default. Available
profiles must be non-empty and unique. When `available` is non-empty, the default,
active profiles, and producer profiles must appear in it. Profile selection
precedence is explicit runtime/CLI value, `MATE_PROFILE`, `default`, then
`production`. Comma-separated active profiles are supported.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `default` | string | no | Profile used when neither an explicit runtime value nor `MATE_PROFILE` is set. |
| `available` | list of strings | no | Profiles allowed for this application. |

---

### `coordinators` {#coordinators-element}

```yaml
coordinators:
  bootstrap:
  - name: initialize
    run_level: 0
    producers: [init]
  schedules:
  - name: every_minute
    cron: "* * * * *"
    producers: [tick]
  routes:
  - name: status
    method: GET
    path: /status
    portal: Status
    producers: [status_page]
```

`coordinators` groups model startup, schedule, and HTTP route coordinators. They
run profile-matching producers in the order listed. An empty `producers: []` is a
valid no-op for every coordinator type and produces `null`; a route then writes an
empty successful response. Otherwise, every referenced producer must exist.
Profile filtering preserves declared order.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `bootstrap` | list of bootstrap objects | no | Startup coordinators. |
| `schedules` | list of schedule objects | no | Cron-triggered coordinators. |
| `routes` | list of route objects | no | HTTP-triggered coordinators. |

---

#### `bootstrap`

```yaml
- name: initialize
  run_level: 0
  producers: [init, warm_cache]
```

A bootstrap coordinator invokes producers during model startup. Bootstrap
coordinators execute in ascending run-level order. A failure prevents higher
levels from starting.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `name` | string | yes | Non-empty name, unique among bootstrap coordinators in its source. |
| `run_level` | integer | no | Startup ordering level; defaults to `0`. |
| `producers` | list of strings | no | Ordered producer names to invoke; omission is an empty list. |

---

#### `schedules`

```yaml
- name: every_minute
  cron: "* * * * *"
  producers: [tick]
```

A schedule coordinator invokes its producers whenever its cron expression fires.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `name` | string | yes | Non-empty schedule name. |
| `cron` | string | yes | Non-empty cron expression. |
| `producers` | list of strings | no | Ordered producer names to invoke; omission is an empty list. |

---

#### `routes`

```yaml
- name: status
  method: GET
  path: /status
  producers: [status_page]
  portal: Status
  remember: [model]
```

A route coordinator exposes an exact HTTP path and invokes producers for matching
requests. Query parameters and form values become execution parameters. String
results are written as text or HTML and other results as JSON. Portal links use
the final namespaced path and are grouped by owning model. Routes use exact paths;
path parameters are not implemented.

The runtime supports model reload and restart operations. Reload preserves the
database connection and share and does not run bootstrap. Model restart reopens
that model's database, clears its share, and runs its bootstrap. Full restart does
this for all models. Failed reloads leave the active model unchanged.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `name` | string | yes | Non-empty route name. |
| `method` | string | yes | Non-empty HTTP method. |
| `path` | string | yes | Non-empty exact route path. |
| `producers` | list of strings | no | Ordered producer names to invoke; omission is an empty list. |
| `portal` | string | no | Non-empty display label that includes the route in portal navigation. |
| `remember` | list of strings | no | Query parameters persisted in HTTP-only cookies and restored when omitted. |

---

### `producers` {#producers-element}

```yaml
producers:
- name: hello
  profiles: [production]
  activities:
  - name: greeting
    compose: |
      root = "hello"
```

`producers` is an ordered list and a top-level YAML composition boundary. Each
entry is a named, reusable sequence of activities.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| list item | producer object | no | A producer definition; names must be unique within the model. |

---

#### Producer element

```yaml
- name: hello
  profiles: [production]
  activities:
  - name: greeting
    compose: |
      root = "hello"
```

A producer returns its last activity's value. A producer without `profiles`
matches every active profile set. A profiled producer matches when at least one
of its profiles is active. Profile names must be non-empty, unique and declared
in `application.profiles.available`.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `name` | string | yes | Non-empty producer name, unique within its model. |
| `profiles` | list of strings | no | Profiles in which this producer is eligible. |
| `activities` | list of activity objects | no | Ordered executable activities. |

---

#### Activity element

```yaml
- name: example
  compose: |
    root = "value"
  state: input
  share: output
```

Every activity has the common `name`, `state`, and `share` properties plus exactly
one supported activity-type property. Activity names are keys below `activity`;
avoid duplicates in one activity context. Nested activities can read their outer
context and earlier values from their own nested list.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `name` | string | yes | Non-empty name used for its result in the activity context. |
| `compose` | string or object | one type required | Runs a Bloblang mapping. |
| `html` | object | one type required | Renders an inline or model-owned Go HTML template. |
| `io` | object | one type required | Invokes a registered IO mechanism. |
| `iterate` | object | one type required | Runs nested activities for every array item. |
| `branch` | object | one type required | Selects one nested activity list. |
| `log` | object | one type required | Emits a runtime log entry. |
| `call` | object | one type required | Calls named producers. |
| `sql` | object | one type required | Executes SQL against the model database. |
| `state` | string | no | State boundary hint: `input`, `output`, or `both`. |
| `share` | string | no | Share boundary hint: `input`, `output`, or `both`. |

---

##### `compose`

```yaml
- name: total
  compose: |
    root = 1 + 2

- name: equivalent
  compose:
    bloblang: |
      root = 1 + 2
  state: input
  share: output
```

`compose` runs a Bloblang mapping; its final `root` becomes the activity value.
The scalar string and object forms are equivalent. Use Bloblang `let` for local
variables and `root = ...` for the result. The example shows both supported forms;
common activity properties are shown on the object form.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `bloblang` | string | yes | Bloblang mapping whose final `root` becomes the activity value. |

---

##### `io`

```yaml
- name: get_json
  io:
    via: fetch
    direction: input
    opts: {}
    params:
      return: json
      url: https://example.com/data.json
  state: input
  share: output
```

`io` invokes a registered IO mechanism. Mechanism options and operation parameters
are evaluated at runtime. Unsupported `via` values fail validation.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `via` | string | yes | Registered mechanism: `fetch` or `tcp`. |
| `direction` | string | no | Boundary metadata: `input`, `output`, or `both`; Boundary View defaults it to `input`. |
| `opts` | map | no | Mechanism-level options; currently unused by built-in mechanisms. |
| `params` | map | no | Operation parameters passed to the mechanism; omission supplies an empty map. |

---

###### Fetch parameters

```yaml
params:
  url: https://example.com/data.json
  return: response
  options:
    method: POST
    headers:
      Content-Type: application/json
    body: '{"enabled":true}'
```

The `fetch` parameter object performs an HTTP request. It defaults to `GET` and
uses no mechanism-level `opts`.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `url` | string | yes | Request URL. |
| `return` | string | no | `json` for decoded JSON, `response` for `{status, headers, body}`, otherwise body text. |
| `options` | object | no | HTTP request options. |
| `options.method` | string | no | HTTP method; defaults to `GET`. |
| `options.headers` | map | no | Request headers converted to strings. |
| `options.body` | string | no | Request body. |

---

###### TCP parameters

```yaml
params:
  server: tcp://example.com:9000
  options:
    body: ping
```

The `tcp` parameter object opens a TCP connection with a ten-second dial timeout,
writes the body, and returns `{sent: true, addr: ...}`.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `server` | string | yes | Server address; an optional `tcp://` prefix is removed. |
| `options` | object | no | TCP operation options. |
| `options.body` | string | no | Bytes written to the connection as a string. |

---

##### `log`

```yaml
- name: audit
  log:
    level: info
    message: 'processed ${params.id}'
    fields:
      request_id: =params.id
    expr: =activity.result.value
  state: input
  share: output
```

`log` emits a runtime log entry and produces `null`. A disabled level skips the
entry without evaluating its message or fields. Output includes a UTC timestamp,
uppercased level, and producer name.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `level` | string | no | Log level; defaults to `info`. |
| `message` | string | no | Literal, `=` expression, or text containing `${...}` interpolation. |
| `fields` | map of strings | no | Structured fields; every value is expression-aware. |
| `expr` | string | no | Expression emitted in a field named `expr`. |

---

##### `sql`

```yaml
- name: save_user
  sql:
    query: INSERT INTO users(name, active) VALUES (?, ?)
    args:
    - =params.name
    - =params.active
  state: input
  share: output
```

`sql` executes a statement against the current model's `model.db`. Row-returning
statements return objects keyed by column; mutations return `rows_affected` and
`last_insert_id`. Names beginning `_sys_` are reserved.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `query` | string | yes | Non-empty SQL statement; expression-aware. |
| `args` | list | no | Expression-aware positional arguments. |

---

##### `iterate`

```yaml
- name: normalized_items
  iterate:
    over: =params.items
    as: entry
    activities:
    - name: normalized
      compose: |
        root = entry.string().uppercase()
  state: input
  share: output
```

`iterate` runs a nested activity list once for every array item.
`iterator.item` and `iterator.index` are available in addition to the binding
named by `as`. The result is an array containing the last nested activity value
for every item. A null input produces `[]`; a non-array fails.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `over` | string | yes | Expression that must produce an array or `null`. |
| `as` | string | no | Name bound to the current item; defaults to `item`. |
| `activities` | list of activities | no | Activities run once for each item; omission supplies an empty list. |

---

##### `branch`

```yaml
- name: access
  branch:
    when: =params.is_admin
    then:
    - name: allowed
      compose: 'root = "allowed"'
    else:
    - name: denied
      compose: 'root = "denied"'
  state: input
  share: output
```

`branch` evaluates a boolean expression and runs one of two nested activity lists.
Its result is the last selected nested activity value, or `null` for an empty
selected list.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `when` | string | yes | Expression that must produce a boolean. |
| `then` | list of activities | no | Activities selected when true. |
| `else` | list of activities | no | Activities selected when false. |

---

##### `html`

```yaml
- name: user_card
  html:
    template: '<strong>{{ .name }}</strong>'
    value: =params.user
  state: input
  share: output
```

`html` renders an inline Go HTML template or a template file owned by the model
and returns an HTML fragment. Without
`value`, dot data is the runtime context map. Go's `html/template` escapes normal
values. Use `{{ html .activity.name.value }}` only for markup produced by an
earlier HTML activity. `template_file` is resolved relative to the model
directory and cannot escape it. Set exactly one of `template` and
`template_file`.

The Mate binary serves the embedded declarative layout stylesheet at `/assets/static/mate-layout.css`. It
provides the documented layout attributes as well as neutral button, form, panel,
card, badge, toolbar, table, alert, and operation-output primitives.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `template` | string | conditional | Non-empty inline Go HTML template. |
| `template_file` | string | conditional | Relative path to a template owned by the model. |
| `value` | any | no | Expression-aware value used as template dot data. |

---

##### `call`

```yaml
- name: notifications
  call:
    mode: many
    run: [email_notification, sms_notification]
    params:
      recipient: =params.user
      message: Account updated
  state: input
  share: output
```

`call` invokes named producers after filtering candidates by active profile. `one`
requires exactly one match and returns its value. `optional` permits zero or one
and returns the value or `null`. `many` permits any number and returns their values
in `run` order. Every named candidate must exist.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `mode` | string | no | Cardinality mode: `one`, `optional`, or `many`; defaults to `one`. |
| `run` | list of strings | yes | Ordered candidate producer names. |
| `params` | map | no | Expression-aware parameters passed to called producers. |

---

##### Boundary hints

```yaml
- name: cached_user
  compose: |
    root = share_get("user")
  state: input
  share: both
```

The common `state` and `share` properties declare inspection metadata and do not
alter execution or the activity value. Without a hint, Boundary View scans
expression text for literal-key calls to the corresponding get, set, init, and
delete helpers. A hint replaces inferred accesses for that resource type.

HTTP resources are grouped by scheme, host, and port. State and Share resources
are identified by key. A hint without a detectable literal key remains semantic
inspection metadata but does not produce a generic resource node.

| Property | Type | Required | Meaning |
| --- | --- | --- | --- |
| `state` | string | no | State boundary hint: `input`, `output`, or `both`. |
| `share` | string | no | Share boundary hint: `input`, `output`, or `both`. |

## External Helpers

This chapter contains the runtime context that was formerly repeated beside individual YAML elements. These names and helpers are external to the YAML schema: YAML fields opt into expression or template evaluation, but do not declare the context itself.

### Runtime Context

Expression-capable activity fields can access:

| Name | Meaning |
| --- | --- |
| `activity.<name>.value` | Result of a prior activity. Nested branch and iterate lists also expose prior values in their nested context. |
| `params.<name>` | Parameters supplied by a coordinator, runtime/CLI invocation, or call activity. |
| `config` | Configuration supplied by the runtime constructing the producer runtime; there is no `config:` YAML schema. |
| `env.<name>` | A process environment value or declared default from `model.env`. |
| `caller` | Information about the current invocation's caller. |
| iterator binding | The current iterate item (`item` by default or the name in `as`). |
| `iterator.item` | Current iterate item. |
| `iterator.index` | Zero-based iterate index. |

Producers receive external inputs through `params`. Call parameters are evaluated in the caller's context before the called producer receives them. There is no separate `input` namespace.

### Expression Evaluation

`compose` is Bloblang directly. For `html.value`, `io.opts`, `io.params`, `sql.query`, `sql.args`, `iterate.over`, `branch.when`, `log` expression fields, and `call.params`, strings beginning with `=` are evaluated and other strings remain literal. Nested maps and arrays are evaluated recursively where the property accepts them. Log messages additionally support `${...}` interpolation.

Profile filtering applies whenever a coordinator or call activity is about to execute a producer. Non-matching producers are skipped without reordering the remaining producers.

Every loaded model has equal access to the complete model-facing runtime API,
including the runtime inspect helpers. Model YAML cannot enable, disable, or
otherwise assign runtime API permissions.

### State Helpers

```bloblang
state_get(key)
state_init(key, initial)
state_set(key, value)
state_list()
state_delete(key)
model_state_get(model, key)
```

| Helper | Meaning |
| --- | --- |
| `state_get` | Returns the stored value and fails when the key is absent. |
| `state_init` | Atomically stores the initial value only when absent, then returns the effective value. |
| `state_set` | Stores and returns a value. |
| `state_list` | Returns entries containing `key`, `value`, and `type`. |
| `state_delete` | Deletes a key and returns a boolean. |
| `model_state_get` | Reads another loaded model's state and fails when the model or key is unavailable. |

These helpers are available to Bloblang and expression evaluation; `state_get` is also an HTML template function. Stored null is an existing value. Use `state_init`, not `state_get(...).or(...)`, to initialize. Cross-model state mutation helpers do not exist.

### Share Helpers

```bloblang
share_get(key)
share_init(key, initial)
share_set(key, value)
share_list()
share_delete(key)
model_share_get(model, key)
```

| Helper | Meaning |
| --- | --- |
| `share_get` | Returns the stored value and fails when the key is absent. |
| `share_init` | Atomically stores the initial value only when absent, then returns the effective value. |
| `share_set` | Stores and returns a value. |
| `share_list` | Returns entries containing `key`, `value`, and `type`. |
| `share_delete` | Deletes a key and returns a boolean. |
| `model_share_get` | Reads another loaded model's share and fails when the model or key is unavailable. |

These helpers are available to Bloblang and expression evaluation; `share_get` is also an HTML template function. Stored null is an existing value. Use `share_init`, not `share_get(...).or(...)`, to initialize. Cross-model share mutation helpers do not exist.

### Activation-local Helpers

Activation data is temporary and isolated to one coordinator activation. All
producers and activities in that activation share it; concurrent and later
activations do not. Use it for request-specific working data and UI view models.

```bloblang
activation_get(key)
activation_set(key, value)
activation_has(key)
activation_delete(key)
```

`activation_get` fails when the key is absent. Operations outside an active
activation fail explicitly. HTML templates read the same store with
`{{ activation "key" }}` and can access nested values, for example
`{{ (activation "view").title }}`.

| Store | Lifetime | Visibility | Persistence |
| --- | --- | --- | --- |
| State | Until changed or its database is replaced | All model activations | Persistent |
| Share | Until cleared or model/application restart | All model activations | In memory |
| Activation | One activation | Producers and activities in that activation | None |

### Metric Activity

The `metric` activity writes one atomic timestamped sample to the current
model's runtime-managed `metrics` table in `model.db`:

```yaml
- name: pool_status
  metric:
    name: pool
    properties:
      temperature: this.temperature
      chlorine: this.chlorine
```

`metric.name` is optional and inherits the activity name when omitted.
`properties` is a required mapping of static property names to Bloblang
expressions. All expressions are evaluated before anything is written. Every
row receives the same generated `sample_id`, UTC Unix-millisecond `timestamp`,
and metric name. Scalars retain their SQLite types, null becomes SQL `NULL`,
and objects or arrays are stored as JSON text.

For one event with dynamic identity and source time, use `property`,
`timestamp`, and `value` together instead of `properties`. `name` may then be
an expression. The property must evaluate to text or an integer, and timestamp
accepts RFC3339 or Unix milliseconds.

Read, aggregate, pivot, bucket, and normalize metric timelines with ordinary
SQL activities. Bootstrap SQL may create reusable views. Mate adds no
metric query language, retention service, or automatic timeline alignment.
See `docs/model-metrics.md` for query examples and the canonical schema.

### Config and Environment Access

`config` is part of the runtime context, but model YAML has no `config:` element. `env.NAME` accesses a variable declared below `model.env`; a process value takes precedence over its default and a declaration without a default requires the process value when used.

### HTML Template Helpers

HTML templates receive the runtime context as dot data unless `html.value` replaces it. Template functions include `html`, `state_get`, `share_get`, `activation`, `value_explorer`, and `markdown_doc`.

Portal navigation is rendered by the portal model from `.config.portal`. It contains loaded routes with a non-empty `portal` property, grouped by owning model.
