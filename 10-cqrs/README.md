# 10. CQRS (Command Query Responsibility Segregation)

> One model to **decide** and another to **read**. Changing data and asking about data become separate problems.

The term was coined by Greg Young, from an older principle by Bertrand Meyer: a method should either change something or
answer something, but not both.

## The idea

In every other folder, the same `Order` struct validates rules, gets saved and gets returned to the client. Those are
three jobs with different needs. Here, they are split:

```text
                 ┌──────────────► write/ ──── decides, and records the facts
  POST (command) │                  │
                 │                  │ events: "OrderPlaced", "StockChanged"...
  client ─► api ─┤                  ▼
                 │                read/ ───── keeps views ready to be read
  GET (query)    └──────────────►
```

## Structure

```text
10-cqrs/
├── main.go                ← connects the two sides
├── architecture_test.go   ← neither side may see the other
├── consistency_test.go    ← shows what happens with a queue between the sides
├── events/events.go       ← the only bridge
├── write/                 ← commands: rules, and only what decisions need
│   ├── handler.go
│   └── pricing.go
├── read/                  ← queries: views in the client's format
│   └── store.go
└── api/api.go             ← POST goes to commands, GET goes to queries
```

## Path of a `POST /orders`

1. `api/api.go`: turns the request into a `write.PlaceOrder` command.
2. `write/handler.go`: applies the rules and returns only an ID, or a refusal.
3. `write/handler.go`: publishes `StockChanged` and `OrderPlaced`.
4. `read/store.go`: updates the views.
5. `api/api.go`: **asks the read side** for the view, and responds.

## What it looks like in practice

**The two models are really different.** The write side keeps only what future decisions need. An order here does not
even store its coupon or subtotal, because no rule uses them again:

```go
type order struct {
	customerID string
	status     string
	lines      []line
	totalCents int
}
```

The read side has what the client wants to see, already in the right format:

```go
type OrderView struct {
	ID            string          `json:"id"`
	CustomerID    string          `json:"customer_id"`
	Status        string          `json:"status"`
```

**Each side is optimized for its own job.** The write side answers R08 without scanning all orders:

```go
active    map[string]int // active orders per customer: answers R08 without scanning
```

**A command does not return data.** It returns an ID or a refusal, and the API then reads the result like any other client:

```go
id, err := commands.PlaceOrder(command)
if err != nil {
	fail(w, err)
	return
}
view, _ := views.Order(id)
```

**The read side can have views that exist nowhere else.** This one is an extra route, beyond the [SPEC](../SPEC.md).
Answering it in other folders would mean scanning orders. Here, it is kept up to date as events arrive. Real output:

```text
$ curl localhost:8080/customers/cust-2/summary
{"customer_id":"cust-2","active_orders":1,"paid_cents":30000}
```

Notifications here are also just one more view:

```go
// R14: notifications are just one more view built from the facts
```

## The catch: eventual consistency

This folder delivers events **synchronously**, in the same call, so reads are always up to date. That is why it passes the
contract, and it is a perfectly valid form of CQRS.

The real benefits show up with separate databases and a queue between the sides, and then a cost appears: the write
succeeded, and the read does not know yet. This test demonstrates it:

```go
if _, ok := views.Product(id); ok {
	t.Fatal("the read side should not know about the product yet")
}
```

The contract suite would fail on that version, because it reads stock right after creating an order, and
[R06](../SPEC.md) expects it to have changed. In practice, this is the user who saves something, reloads the page and does
not see it.

## CQRS is not Event Sourcing

They are often mentioned together, but they are independent:

| | CQRS | Event Sourcing |
|---|---|---|
| What it separates | Writing from reading | Nothing: it changes what gets stored |
| What gets stored | The current state | The list of everything that happened |
| In this folder | Yes | No: events update the views and are then discarded |

## How it is tested

Each side is tested alone. The write side by what it decides and publishes:

```go
if got := published.published[1].(events.OrderPlaced); got.DiscountCents != 6_000 || got.TotalCents != 54_000 {
```

The read side by feeding it facts, with no rules or commands involved:

```go
store.Publish(events.OrderPlaced{ID: "o", CustomerID: "c", TotalCents: 500})
```

## The gain and the price

| You gain | You pay |
|---|---|
| Each model is simple, because it does one job | Two models to maintain, plus the events between them |
| Views shaped for each screen, with no complex queries | Changing a field means changing both sides |
| Reads and writes can scale separately | With a queue, eventual consistency |
| The write side is free of display concerns | A lot of machinery for a simple CRUD |

## When to use it

Martin Fowler's warning on the subject is worth keeping in mind: for most systems, CQRS adds risky complexity.

- **Good for:** far more reads than writes, screens that combine a lot of data, and complex rules on the write side.
- **Avoid when:** reading and writing look alike, which is most CRUDs. Also avoid applying it to a whole system: use it only on the part that needs it.

## How to run

```bash
go run .
curl localhost:8080/customers/cust-1/summary
go test ./...
```

**Measurement:** 10 files, 605 lines of code and 5 packages. See the [full comparison](../RESULTS.md).

**Next step:** [11. Vertical Slice](../11-vertical-slice/README.md).
