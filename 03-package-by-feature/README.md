# 03. Monolith by feature (package by feature)

> The code is split by **business subject**: everything about orders lives in the orders folder.

It is also known as a *monolith organized by module*. Beware of the trap in that name:
**having folders by subject does not yet make a modular monolith**. This folder and
[folder 04](../04-modular-monolith/README.md) look the same at first glance, and the difference between them is the
main lesson here.

## Structure

```text
03-package-by-feature/
├── main.go
├── products/
│   └── products.go        ← product type, rules, storage and HTTP
├── customers/
│   └── customers.go
├── orders/
│   └── orders.go
├── notifications/
│   └── notifications.go
├── payments/
│   └── payments.go
└── web/
    └── web.go             ← only what is common: JSON and HTTP errors
```

Compare with folder 02: there the tree said "handler, service, repository".
Here it says "products, customers, orders". **Just by looking at the folders, you know what the system is about.**

## The dependencies

```text
             main
               │
     ┌─────────┼─────────────┐
     ▼         ▼             ▼
  products  customers   notifications
     ▲         ▲             ▲
     └─────────┼─────────────┘
               │
            orders ─────► payments

  (all use web)
```

## Path of a `POST /orders`

1. `orders/orders.go`, `Register` function: receives the HTTP request.
2. `orders/orders.go`, `Place` method: validates and calculates.
3. Reads and writes stock **directly** in `products.Store`.
4. Writes the notification **directly** in `notifications.Store`.

Almost everything is in one file, which is great for understanding a subject. The problem is in steps 3 and 4.

## What it looks like in practice

Cohesion improved a lot: changing a product rule affects `products/` and nothing else. But look at the imports of orders:

```go
import (
	"fmt"
	"net/http"
	"sync"

	"byfeature/customers"
	"byfeature/notifications"
	"byfeature/payments"
	"byfeature/products"
	"byfeature/web"
)
```

And look at how stock is changed. It is the **orders** module deciding how stock changes, using the **products** struct:

```go
product.Stock -= item.Quantity
reserved = append(reserved, product)
```

```go
for _, product := range reserved {
	s.products.Save(product)
}
```

Nothing prevents this, because everything in `products` is exported. The result:

- To change the structure of `products.Store`, you have to check `orders`.
- The stock rule does not live in `products`: it lives wherever it is being used.
- If another subject also touches stock, the rule gets duplicated.

In other words, **the folder is cohesive, but the coupling remains**. Folders organize reading, but they do not impose
boundaries.

## How it is tested

Testing orders requires the concrete implementations of the other features:

```go
productStore, customerStore := products.NewStore(), customers.NewStore()
service := NewService(productStore, customerStore, notifications.NewStore())
```

## The gain and the price

| You gain | You pay |
|---|---|
| The structure tells what the system does | Boundaries are only a convention |
| Each change tends to affect one folder | Features couple without anyone noticing |
| Less jumping between folders (444 lines, the fewest after spaghetti) | Without care, it becomes a confusing dependency graph |
| Easy to split responsibilities between people | In large projects, import cycles appear (in Go, they are compile errors) |

## When to use it

- **Good for:** most small and medium systems. It is a better default than pure layers, and costs almost the same.
- **Avoid when:** several teams share the code or there are plans to extract parts later. In those cases, go to folder 04.

You can also have layers inside each folder. They are not mutually exclusive: `orders/handler.go`,
`orders/service.go` and `orders/repository.go` is a common and healthy organization as a folder grows.

## How to run

```bash
go run .
go test ./...
```

**Measurement:** 8 files and 444 lines of code. See the [full comparison](../RESULTS.md).

**Next step:** [04. Modular monolith](../04-modular-monolith/README.md), which turns these conventions into real boundaries.
