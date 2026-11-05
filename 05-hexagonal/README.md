# 05. Hexagonal architecture (ports and adapters)

> The business sits at the center and does not know technology exists. Everything outside connects to it through **sockets**.

Created by Alistair Cockburn. The name is misleading: there is nothing special about having six sides.
The drawing just needed enough sides to fit several connections.

## The turning point compared to folder 02

This is the whole idea, and it is worth pausing to understand it:

```text
  Layers (folder 02)                Hexagonal (this folder)

     HTTP                           HTTP ──────┐
      │                                        ▼
      ▼                                  ┌──────────┐
   business                              │ business │
      │                                  │ (ports)  │
      ▼                                  └──────────┘
   database                                    ▲
                                   database ───┘
```

In folder 02, the business depends on the database. Here, **the database depends on the business**. The core declares
"I need something that stores orders" (an interface), and the database adapter implements it.
This is called **dependency inversion**.

## Structure

```text
05-hexagonal/
├── main.go                         ← picks the adapters and plugs them in
├── architecture_test.go            ← ensures dependencies point inward
└── internal/
    ├── core/                       ← THE HEXAGON
    │   ├── domain/                 ← rules in plain Go: Order, Product, Coupon
    │   ├── ports/ports.go          ← the sockets (interfaces)
    │   └── app/shop.go             ← the sequence of the use cases
    └── adapters/                   ← THE OUTSIDE WORLD
        ├── in/httpapi/             ← inbound: HTTP calls the core
        └── out/
            ├── memory/             ← outbound: data in memory
            ├── jsonfile/           ← outbound: data in a file
            └── fakepayment/        ← outbound: simulated gateway
```

## Two kinds of socket

| | **Inbound** port (driving) | **Outbound** port (driven) |
|---|---|---|
| Who starts the conversation | The world calls the core | The core calls the world |
| Example here | `ports.Shop` | `ports.OrderRepository`, `ports.PaymentGateway` |
| Adapter here | `httpapi` (or a test) | `memory`, `jsonfile`, `fakepayment` |

The interfaces live with the core, not with whoever implements them:

```go
type OrderRepository interface {
	AddOrder(o domain.Order) domain.Order
	FindOrder(id string) (domain.Order, bool)
	SaveOrder(o domain.Order)
	HasActiveOrder(customerID string) bool
}
```

```go
type PaymentGateway interface {
	Charge(amountCents int) bool
}
```

## The proof: swapping the database without touching the core

This is the architecture's promise, and here it is tested. `jsonfile` implements the same ports as `memory`, and the
choice is made only in `main.go`:

```go
if os.Getenv("STORE") == "json" {
```

```go
shop := app.NewShop(store, store, store, store, fakepayment.New())
```

Measured result, from `scripts/verify.sh`:

```text
== 05-hexagonal
  [ok] contract: in memory (25 tests)
  [ok] contract: JSON file adapter (25 tests)
```

**Not one line of `core/` changed for this.** And the data really survives a restart:

```text
$ STORE=json go run .            # create a product; then restart the process
$ curl localhost:8080/products/prod-1
{"id":"prod-1","name":"Keyboard","price_cents":100,"stock":1}
```

## Path of a `POST /orders`

1. `adapters/in/httpapi`: decodes the JSON and calls the port `shop.PlaceOrder`.
2. `core/app/shop.go`: coordinates the steps, without deciding rules.
3. `ports.CustomerRepository` and `ports.ProductRepository`: fetch the data (without knowing where it is).
4. `core/domain/order.go`: `domain.NewOrder` calculates discount, shipping and total.
5. Outbound ports: save.
6. `httpapi`: converts the domain into a DTO and responds.

## What it looks like in practice

**The domain is rich.** In folder 02, an order was a bag of fields. Here, it protects its own rules:

```go
func (o *Order) Cancel() error {
	// R11
	if o.Status == StatusCancelled {
		return Fail(CodeInvalidTransition, "order already cancelled")
	}
	o.Status = StatusCancelled
	return nil
}
```

**The domain is clean.** No JSON tags, no SQL, no HTTP. The API format lives in the adapter:

```go
type Order struct {
	ID            string
	CustomerID    string
	Status        Status
```

**The rule became a test.** If the domain imports `net/http`, the build breaks (and this was tested):

```go
"internal/core/domain": {"hexagonal/internal", "net/http", "encoding/json", "database/sql", "os"},
```

An honest caveat: the fields of `Order` are exported so adapters can store them. Technically, you can still write
`order.Status = ...` from outside. It is a common compromise in Go; sealing it off would require private fields and
conversion functions in each adapter.

## How it is tested

This is the gain most felt day to day. A fake gateway fits in three lines:

```go
type declineAll struct{}

func (declineAll) Charge(int) bool { return false }
```

So the "payment declined" path is tested with no magic value and no real service. And the pure rules do not even need
that:

```go
order, err := NewOrder("c", items, "WELCOME10", true)
```

## The gain and the price

| You gain | You pay |
|---|---|
| Rules testable with no infrastructure | More types: domain, DTO, port and adapter |
| Swapping technology affects only the adapter | Repetitive conversions between domain and DTO |
| The domain reads without technical noise | Abstraction that may never be used ("what if we switch databases?") |
| Postponing technology decisions | Harder to understand for first-time readers |

A word of caution: **the real reason is rarely switching databases.** That hardly ever happens. The real reason is to
test the rule without the database and keep it readable. If the system is just a CRUD, with no rules, all this
plumbing does not pay for itself.

## Hexagonal does not exclude the others

It is an organization **inside** a unit. You can have a modular monolith with each module hexagonal, or a microservice
that is hexagonal inside. The axes are independent: see the [main README](../README.md).

## When to use it

- **Good for:** domains with important rules, integrations that change, long-lived systems.
- **Avoid when:** the system is a simple CRUD, it is a prototype, or the team has not yet mastered the pattern.

## How to run

```bash
go run .                                    # in memory
STORE=json DATA_FILE=/tmp/d.json go run .   # in a file
go test ./...
```

**Measurement:** 16 files and 841 lines of code. See the [full comparison](../RESULTS.md).

**Next step:** [06. Clean Architecture](../06-clean-architecture/README.md), the same idea with more ceremony.
