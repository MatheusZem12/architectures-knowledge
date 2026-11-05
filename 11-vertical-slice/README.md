# 11. Vertical Slice

> Each feature is one slice, from the HTTP request down to the data. Slices do not share layers, and they do not know each other.

Popularized by Jimmy Bogard. Folder 02 cuts the system horizontally, so a feature crosses every layer. This one cuts
vertically, so a feature stays in one place.

## Structure

```text
11-vertical-slice/
├── main.go                     ← the list of features, one per line
├── architecture_test.go        ← no slice may import another
├── store/store.go              ← the shared data, and no rule at all
├── web/web.go                  ← small HTTP helpers
└── features/
    ├── createproduct/slice.go
    ├── getproduct/slice.go
    ├── createcustomer/slice.go
    ├── getcustomer/slice.go
    ├── listnotifications/slice.go
    ├── placeorder/slice.go     ← the richest, with all the pricing rules
    ├── getorder/slice.go
    ├── payorder/slice.go       ← the payment gateway lives here
    └── cancelorder/slice.go
```

## The two cuts

```text
   Layers (02)                       Vertical slices (11)

                 place   pay                   place   pay
                 order  order                  order  order
   handlers    ───┼──────┼───                 ┌─────┐ ┌─────┐
   services    ───┼──────┼───                 │     │ │     │
   repositories ──┼──────┼───                 │     │ │     │
                                              └──┬──┘ └──┬──┘
                                                 ▼       ▼
                                                 shared data
```

## Compared to the neighbors

| | Layered (02) | Clean (06) | Vertical slice (11) |
|---|---|---|---|
| Unit of organization | The layer | The use case, in rings | The feature |
| Change one feature | Opens several folders | Opens several rings | Opens one folder |
| Interfaces | Few | Many | None here |
| Every feature follows the same recipe? | Yes | Yes | No: each is as complex as it needs |
| Lines of code | 592 | 891 | 500 |

Folders 06 and 11 both have one file per use case. The difference is that 06 still divides each use case across rings,
while here it stays together. This folder is also close to [folder 03](../03-package-by-feature/README.md), which groups
by subject: a slice is much finer, one action rather than a whole "orders" package.

## What it looks like in practice

**The main file reads like a list of what the system does:**

```go
placeorder.Register(mux, db)
getorder.Register(mux, db)
payorder.Register(mux, db)
cancelorder.Register(mux, db)
```

**A simple feature stays simple.** Reading a product has no service, no repository and no DTO. Compare it with the
eight steps of the same request in folder 06:

```go
found, ok := db.Products[r.PathValue("id")]
if !ok {
	return web.Fail("not_found", "product not found")
}
```

**Rules live where they are used.** Pricing is only needed when placing an order, so it lives in that slice:

```go
func discountFor(coupon string, subtotal int, isFirstOrder bool) (int, error) {
```

## The price: repetition

Slices are not allowed to reuse each other, so finding an order appears in `getorder`, `payorder` and `cancelorder`,
almost word for word. This is deliberate. The risk it avoids is a shared abstraction that every feature depends on and
nobody dares to change.

But it can go too far. If a real rule gets duplicated, such as how to calculate a total, the copies will drift apart.
Repeated *lookups* are fine; repeated *business rules* are a sign that something belongs in a shared place.

The other weak spot is the shared data. `store` is the one thing all slices touch, and renaming a field there affects
all of them. A pragmatic choice was made here: those types carry JSON tags and are returned directly. A stricter version
would give each slice its own response type, with more repetition.

## How it is tested

A slice is tested alone. The only setup is the data it reads:

```go
db := store.New()
db.Products["p"] = store.Product{ID: "p", PriceCents: 30_000, Stock: 5}
db.Customers["c"] = store.Customer{ID: "c"}
```

## The gain and the price

| You gain | You pay |
|---|---|
| A change tends to touch one folder | Repeated code between slices |
| Each feature is only as complex as it needs | Less uniformity, so the team needs judgment |
| New features do not risk the old ones | Rules that are really shared need a deliberate home |
| Few abstractions to learn | Everything still depends on the shared data |

## When to use it

- **Good for:** systems where features change independently, such as most business applications, and teams that are tired of ceremony.
- **Avoid when:** many rules are truly shared across features. A rich domain model, as in [folder 05](../05-hexagonal/README.md), handles that better.

## How to run

```bash
go run .
go test ./...
```

**Measurement:** 14 files, 500 lines of code and 12 packages. See the [full comparison](../RESULTS.md).

**Next step:** [12. Event Sourcing](../12-event-sourcing/README.md).
