# 04. Modular monolith

> A single process, but with modules that **only talk through a public API**. The boundary is enforced, not agreed upon.

It is still a monolith: one binary, one deploy, function calls. The difference from
[folder 03](../03-package-by-feature/README.md) is that here a module **cannot** touch another's internals.

## Structure

```text
04-modular-monolith/
├── main.go                         ← the only place that knows all modules
├── architecture_test.go            ← the official dependency map, as a test
├── platform/                       ← common infrastructure, no business rules
│   ├── events/bus.go
│   └── web/web.go
└── modules/
    ├── catalog/
    │   ├── api.go                  ← the module's public API
    │   └── internal/store/         ← private: the compiler blocks outsiders
    ├── customers/
    ├── notifications/
    ├── orders/
    │   ├── api.go
    │   └── internal/
    │       ├── pricing/            ← pure pricing rule
    │       └── store/
    └── payments/
```

## 03 and 04, side by side

This table is the central point of this folder:

| Question | 03: by feature | 04: modular |
|---|---|---|
| Can orders read the product struct? | Yes, all of it | No: it only sees what `catalog/api.go` exposes |
| Who decides how stock goes down? | Orders, by editing the product | Catalog, through `Reserve` |
| What prevents a violation? | Nothing (code review, if someone notices) | The compiler and a test |
| How does orders tell notifications? | Calls its store directly | Publishes an event, without knowing who listens |

## The dependencies

```text
                    main
                      │  assembles everything
     ┌────────┬───────┼─────────┬──────────────┐
     ▼        ▼       ▼         ▼              ▼
  catalog customers payments  orders ◄───── notifications
     ▲        ▲       ▲         │    (listens to the events)
     └────────┴───────┴─────────┘
         orders only uses public APIs
```

## Path of a `POST /orders`

1. `orders/api.go`: validates the items.
2. `customers.Exists(id)`: asks the customers module.
3. `catalog.Reserve(lines)`: requests the reservation, and the catalog decides (all or nothing).
4. `pricing.Discount` and `pricing.Shipping`: pure rules, internal to orders.
5. Saves to its own store and publishes `orders.Event`.
6. `notifications` reacts to the event, without `orders` knowing it exists.

## What it looks like in practice

**The boundary is real.** If `orders` tries to import the catalog's store, Go refuses to compile, because packages in
`internal/` are only visible inside the parent folder. This was actually tested:

```text
use of internal package modular/modules/catalog/internal/store not allowed
```

**Stock only changes by asking.** Compare with folder 03. Orders no longer sees the `Stock` field:

```go
// R06 and R07: the catalog reserves all or nothing and returns the current price
reserved, err := m.catalog.Reserve(lines)
if err != nil {
	return store.Order{}, err
}
```

**Event-based decoupling** means deleting the notifications module does not require changing `orders`:

```go
func New(bus *events.Bus) *Module {
	m := &Module{store: store.New()}
	bus.Subscribe(m.onEvent)
	return m
}
```

**The dependency map is an explicit decision.** `internal/` protects the inside of modules, but does not stop a module
from calling the public API of one it should not. This test covers that case, and breaks the build:

```go
var allowed = map[string][]string{
	"catalog":       {},
	"customers":     {},
	"payments":      {},
	"orders":        {"catalog", "customers", "payments"},
	"notifications": {"orders"},
}
```

This kind of test is called a *fitness function*: an architecture rule that runs as a test.

## An important consequence: compensation already appears

The catalog reserves stock before the coupon is evaluated. If the coupon is refused, the reservation must be undone:

```go
// R08: a refused coupon undoes the reservation (compensation)
order.DiscountCents, err = pricing.Discount(in.Coupon, order.SubtotalCents, !m.store.HasActiveOrder(in.CustomerID))
if err != nil {
	m.catalog.Release(lines)
	return store.Order{}, couponError(err)
}
```

In folder 02, everything happened with direct access to the data, and nothing was saved before the end. Separating data
owners brings this cost even inside the monolith. It is the same idea as the saga in
[folder 07](../07-microservices/README.md), only without the network: here, the failure between the two steps does not
come from a dropped connection.

## How it is tested

The pricing rule became pure logic, testable with nothing around it:

```go
got, err := Discount(c.coupon, c.subtotal, c.first)
```

## The gain and the price

| You gain | You pay |
|---|---|
| Boundaries enforced by the compiler | More code: an API and its own types per module |
| Modules that evolve independently | Thinking hard about what each module exposes |
| Operational simplicity: a single deploy | No fault isolation: a memory leak or a `panic` in a goroutine takes everything down |
| An open path to extracting a microservice | No independent scaling per module |

## When to use it

It is often the **best starting point for a serious system**: it gives you the boundaries of microservices without the
cost of the network. If a module ever has to leave, its public API is already the new service's contract.

- **Good for:** products that will grow, one or a few teams, domains that are reasonably well understood.
- **Avoid when:** the system is small (folder 03 is enough) or the modules truly need independent deploys and scaling.

For more rigor, each module can have its own tables or schema, and never `JOIN` across modules. Without that, the
database becomes the back door that bypasses every boundary.

## How to run

```bash
go run .
go test ./...         # includes the dependency test
```

**Measurement:** 15 files, 678 lines of code and 13 packages. See the [full comparison](../RESULTS.md).

**Next step:** [05. Hexagonal](../05-hexagonal/README.md). So far, we looked at boundaries *between subjects*.
Now the focus moves to the boundary *between business and technology*.
