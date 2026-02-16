# gobusgen

A code generator that produces type-safe, in-process event bus implementations from Go map declarations.

## Installation

```bash
go install github.com/hay-kot/gobusgen@latest
```

## Quick Start

Define your events as a `map[string]any` variable where keys are event names and values are payload struct literals:

```go
package events

//go:generate gobusgen generate

type UserCreatedEvent struct {
    UserID string
    Email  string
}

type OrderPlacedEvent struct {
    OrderID string
    Total   float64
}

var Events = map[string]any{
    "user.created": UserCreatedEvent{},
    "order.placed": OrderPlacedEvent{},
}
```

Run the generator:

```bash
gobusgen generate
```

This produces an `eventbus.gen.go` file with typed publish/subscribe methods:

```go
bus := events.New(128) // buffered channel size

go bus.Start(ctx)

bus.SubscribeUserCreated(func(e events.UserCreatedEvent) {
    fmt.Println("new user:", e.Email)
})

bus.PublishUserCreated(events.UserCreatedEvent{
    UserID: "123",
    Email:  "user@example.com",
})
```

## Usage

```
gobusgen generate [-p <dir>.<Var>] [-o <file>]
```

| Flag            | Description                                                                            |
| --------------- | -------------------------------------------------------------------------------------- |
| `-p, --package` | Target as `<dirpath>.<VarName>` (default: `.Events`). Repeatable for multiple targets. |
| `-o, --output`  | Output file path. Only valid with a single `--package` target.                         |

```bash
# Generate from ./Events in current directory
gobusgen generate

# Generate from a specific package and variable
gobusgen generate -p ./events.Events

# Write to a specific output file
gobusgen generate -p ./events.Events -o ./events/bus.gen.go

# Generate from multiple sources
gobusgen generate -p ./events.Events -p ./commands.Commands
```

Output defaults to `<dir>/<prefix>bus.gen.go` where the prefix is derived from the variable name (`Events` -> `eventbus.gen.go`, `Commands` -> `commandbus.gen.go`).

## Event Map Keys

Map keys can be:

- String literals: `"user.created"`
- Bare constants: `UserCreatedKey`
- String conversions: `string(TypedConst)`
- Const aliases referencing other string constants

Event names may contain letters, digits, dots, hyphens, and underscores.

## Prefix Directive

The prefix used for generated type names is derived from the variable name. `Events` produces no prefix, `OrderEvents` produces `Order`, and `Commands` produces `Command`.

Override the prefix with the `//gobusgen:prefix` directive:

```go
//gobusgen:prefix Notification
var Commands = map[string]any{ ... }
```

An empty directive (`//gobusgen:prefix`) produces no prefix. The directive may appear above the `var` keyword or above the variable name inside a grouped `var()` block.

## Generated Event Bus

The generated code provides:

- **Typed constants** for each event name (`EventUserCreated Event = "user.created"`)
- **Typed publish methods** (`PublishUserCreated(UserCreatedEvent)`)
- **Typed subscribe methods** (`SubscribeUserCreated(func(UserCreatedEvent))`)
- **Non-blocking publish** via buffered channels — events are dropped if the buffer is full
- **Panic recovery** — subscriber panics are caught and reported, not propagated
- **Concurrency safety** — thread-safe publish and subscribe with `sync.RWMutex`

### Lifecycle Hooks

```go
bus.OnPublish(func(event Event, payload any) {
    // fires after an event is successfully enqueued
})

bus.OnDrop(func(event Event, payload any) {
    // fires when an event is dropped due to a full buffer
})

bus.OnSubscribe(func(event Event) {
    // fires when a subscriber is registered
})

bus.OnPanic(func(event Event, payload any, recovered any) {
    // fires when a subscriber panics
})
```

## License

[MIT](LICENSE)
