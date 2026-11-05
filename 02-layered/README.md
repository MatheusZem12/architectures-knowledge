# 02. Layered monolith (by technical responsibility)

> The code is split by the **type of work** it does: what speaks HTTP, what decides, what stores.

This is the most common structure in the world, and the default of many frameworks (such as `controller`, `service`
and `repository` in Spring). It is also known as a *monolith by responsibility*.

## Structure

```text
02-layered/
├── main.go                     ← assembles everything
└── internal/
    ├── handler/                ← presentation: HTTP and JSON
    ├── service/                ← business: all the rules
    ├── repository/             ← persistence: stores and fetches
    ├── integration/            ← external systems (payment)
    └── model/                  ← structs used by all layers
```

## The rule: each layer only knows the one below

```text
   handler
      │
      ▼
   service  ───────┐
      │            │
      ▼            ▼
  repository   integration

  (model is used by all)
```

The arrows point down all the way: **the business depends on persistence**.
That is exactly what [folder 05](../05-hexagonal/README.md) will invert.

## Path of a `POST /orders`

1. `handler/router.go`: decodes the JSON and calls `orders.Place`.
2. `service/order_service.go`: validates, calculates and decides.
3. `repository/*.go`: reads customers and products, saves stock and the order.
4. `handler/router.go`: turns the result (or the error) into an HTTP response.

## What it looks like in practice

The separation works. Newcomers know where to look: the rule is in the service, the query is in the repository.

But look at `OrderService`. This is the classic symptom of this architecture:

```go
type OrderService struct {
	mu            sync.Mutex // in a real database, this would be a transaction
	orders        *repository.OrderRepository
	products      *repository.ProductRepository
	customers     *repository.CustomerRepository
	notifications *repository.NotificationRepository
	payments      *integration.PaymentGateway
}
```

There are two problems in these lines:

1. **The service knows everything.** Over time, it tends to become the new giant file: a "god service".
2. **The types are concrete.** `*repository.OrderRepository` is a struct, not an interface. The business is tied to the persistence implementation.

The model is also anemic: `model.Order` only has fields, and anyone can write `order.Status = "paid"` from anywhere.
The rule forbidding paying twice lives in the service, and nothing forces you to go through it:

```go
type Order struct {
	ID            string      `json:"id"`
	CustomerID    string      `json:"customer_id"`
	Status        string      `json:"status"`
```

The `json` tags on the model show another leak: changing the API format alters the struct the business uses.

## How it is tested

To test a discount, you must assemble the real repositories and gateway:

```go
service := NewOrderService(
	repository.NewOrderRepository(), products, customers,
	repository.NewNotificationRepository(), integration.NewPaymentGateway(),
)
```

It is cheap here because everything is in memory. With a real database, this test would start to require the database,
or you would have to create interfaces just for the test.

## The gain and the price

| You gain | You pay |
|---|---|
| A structure everyone recognizes | A feature gets scattered across several folders |
| An obvious place for each type of code | The service tends to grow without limit |
| Testing persistence and HTTP separately | The business depends on persistence |
| A quick start | Nothing stops `service` from using any repository |

The price most felt day to day is the first line: to change "orders", you open a file in `handler/`, one in
`service/`, one in `repository/` and one in `model/`. The folders tell **how** the system is built, not **what** it
does. [Folder 03](../03-package-by-feature/README.md) tackles exactly that.

## When to use it

- **Good for:** CRUDs, small and medium systems, teams that already know the pattern, MVPs that must ship fast.
- **Avoid when:** the domain has many rules, several teams will work at once, or the system must switch infrastructure easily.

## How to run

```bash
go run .              # port 8080
go test ./...         # its own tests
```

**Measurement:** 14 files and 592 lines of code. See the [full comparison](../RESULTS.md).

**Next step:** [03. Package by feature](../03-package-by-feature/README.md).
