# 06. Clean Architecture

> Concentric rings, where dependencies only point inward. **One use case per file.**

Proposed by Robert C. Martin ("Uncle Bob").

First, a clarification: **Clean Architecture is not the same as Clean Code.** The names are similar, but the first is
about the structure of the system, and the second about how each function is written.
Clean Code is in [`principles/clean-code`](../principles/clean-code/README.md).

## The rings

```text
  ┌───────────────────────────────────────────┐
  │ frameworks/       net/http, main          │
  │  ┌─────────────────────────────────────┐  │
  │  │ adapters/    controllers, presenters│  │
  │  │              gateways               │  │
  │  │  ┌───────────────────────────────┐  │  │
  │  │  │ usecases/    what the system  │  │  │
  │  │  │              does             │  │  │
  │  │  │  ┌─────────────────────────┐  │  │  │
  │  │  │  │ entities/   enterprise  │  │  │  │
  │  │  │  │             rules       │  │  │  │
  │  │  │  └─────────────────────────┘  │  │  │
  │  │  └───────────────────────────────┘  │  │
  │  └─────────────────────────────────────┘  │
  └───────────────────────────────────────────┘

  Dependency rule: code may only point inward.
```

## Structure

```text
06-clean-architecture/
├── main.go
├── architecture_test.go
├── entities/                ← order.go, product.go, coupon.go...
├── usecases/
│   ├── gateways.go          ← the interfaces the use cases need
│   ├── dto.go               ← the data that leaves the use cases
│   ├── place_order.go
│   ├── pay_order.go
│   ├── cancel_order.go
│   └── ...                  ← one file per thing the system does
├── adapters/
│   ├── controllers/         ← input, without net/http
│   ├── presenters/          ← output format (JSON)
│   └── gateways/            ← memory and payment
└── frameworks/
    └── web/server.go        ← the only file that knows net/http
```

A simple `ls usecases/` lists what the system does. Uncle Bob calls this **screaming architecture**: the structure
talks about the business, not the framework.

## So what is the difference from hexagonal?

Smaller than it seems. **Both follow the same rule: dependencies point to the center.** Many people use the terms as
synonyms. What changes here is the granularity:

| | Hexagonal (05) | Clean (06) |
|---|---|---|
| Orchestration | `app/shop.go` with several methods | One type and one file per use case |
| Interfaces | Per resource (`OrderRepository`) | Per capability (`OrderReader`, `OrderSaver`) |
| Output | The domain reaches the HTTP adapter | DTOs keep the entity from leaving the ring |
| Conversion for the client | Inside the HTTP adapter | In a separate presenter |
| HTTP | The adapter uses `net/http` | The controller does not even know `net/http` |
| Files / lines of code | 16 / 841 | 25 / 891 |

## Path of a `POST /orders`

1. `frameworks/web/server.go`: receives the HTTP request and builds a simple `controllers.Request`.
2. `adapters/controllers`: converts the JSON into `usecases.PlaceOrderInput`.
3. `usecases/place_order.go`: runs the use case, in a transaction.
4. `entities/order.go`: calculates amounts and validates rules.
5. `adapters/gateways`: implements the persistence interfaces.
6. `usecases`: returns an `OrderOutput` (the entity does not leave here).
7. `adapters/presenters`: turns it into output JSON.
8. `frameworks/web`: writes the HTTP response.

That is eight steps for what, in folder 01, fit in an `if`. That is the cost, in plain sight.

## What it looks like in practice

**Each use case declares exactly what it needs.** Looking at the struct, you already know everything `PayOrder` touches:

```go
type PayOrder struct {
	orders        OrderSaver
	notifications NotificationWriter
	payments      PaymentGateway
	tx            Transactor
}
```

**The interfaces are minimal**, a direct application of the interface segregation principle (the "I" in SOLID):

```go
type OrderReader interface {
	FindOrder(id string) (entities.Order, bool)
}

type OrderSaver interface {
	OrderReader
	SaveOrder(entities.Order)
}
```

**The transaction is an abstraction.** The use case knows it needs atomicity, but not how it is achieved:

```go
type Transactor interface {
	Atomic(fn func() error) error
}
```

**The controller does not know HTTP.** A simple `Request` goes in and a simple `Response` comes out:

```go
func (c *Controller) PayOrder(r Request) Response {
	out, err := c.uc.PayOrder.Execute(r.PathID)
	return result(200, presenters.PresentOrder(out), err)
}
```

Replacing `net/http` with another framework rewrites `frameworks/web/server.go` and the line in `main.go` that starts
the server.

A note on the "pure" version: in the book, the use case calls a presenter through an interface, instead of returning a
value. In Go, returning a DTO is more natural and keeps the same dependency direction, so that is the option used here.

## How it is tested

Thanks to the small interfaces, the fake only needs the methods the use case really uses:

```go
type fakeOrders struct{ order entities.Order }

func (f *fakeOrders) FindOrder(string) (entities.Order, bool) { return f.order, true }
func (f *fakeOrders) SaveOrder(o entities.Order)              { f.order = o }
```

Compare with what a "repository of everything" would require, or see the ISP example in
[`principles/solid`](../principles/solid/README.md).

## The gain and the price

| You gain | You pay |
|---|---|
| The list of use cases is visible in the tree | The largest volume of code and files in the repository |
| Each use case is isolated and easy to test | Lots of conversion: Input, Output, View |
| Swapping the web framework only affects the outer ring | Eight steps to cross one request |
| Dependency rule verified by a test | Easy to apply as dogma to systems that do not need it |

## When to use it

- **Good for:** large long-lived systems, with many rules and use cases, and large teams that benefit from a rigid standard.
- **Avoid when:** the project is small. Hexagonal delivers nearly the same with less ceremony, and is usually the better starting point. In practice, few systems need all four rings to the letter.

## How to run

```bash
go run .
go test ./...
```

**Measurement:** 25 files and 891 lines of code. See the [full comparison](../RESULTS.md).

**Next step:** [07. Microservices](../07-microservices/README.md). So far, everything was a single process.
