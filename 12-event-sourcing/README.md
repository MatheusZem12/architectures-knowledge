# 12. Event Sourcing

> Store **everything that happened**, never the current state. The state is calculated by replaying the facts.

Think of a bank statement. The balance is not the source of truth: it is the result of adding up all the transactions.
Every other folder here stores the balance and forgets the transactions. This one does the opposite.

## Structure

```text
12-event-sourcing/
├── main.go
├── events/events.go        ← everything that can happen, in the past tense
├── eventstore/store.go     ← an append-only log: no update and no delete
├── domain/domain.go        ← replays events into state
├── commands/commands.go    ← replay, check the rules, append new events
├── projections/views.go    ← disposable read models, built from the log
└── api/api.go
```

## The flow of a command

```text
  1. load        events of this product, from the log
  2. replay      ProductCreated(5) + StockReserved(1) = stock 4
  3. decide      is there enough stock?
  4. append      StockReserved(1), a new fact at the end of the log
  5. project     the views update from the new fact
```

## What it looks like in practice

**State is a calculation.** This is the heart of the architecture:

```go
func ReplayProduct(history []any) Product {
	var product Product
	for _, event := range history {
		switch e := event.(type) {
		case events.ProductCreated:
			product = Product{Exists: true, PriceCents: e.PriceCents, Stock: e.Stock}
		case events.StockReserved:
			product.Stock -= e.Quantity
		case events.StockReleased:
			product.Stock += e.Quantity
		}
	}
	return product
}
```

**Nothing is ever undone.** Cancelling does not edit the past. It adds new facts:

```go
// R13: nothing is undone. A new fact says the stock came back.
```

## See it running

**The complete history of anything**, through an extra route that no other folder can offer. This is real output, for a
product that was reserved by an order that was later cancelled:

```text
$ curl localhost:8080/products/prod-1/history
{"seq":1,"type":"ProductCreated","data":{"ProductID":"prod-1","Name":"Keyboard","PriceCents":30000,"Stock":5}}
{"seq":3,"type":"StockReserved","data":{"ProductID":"prod-1","OrderID":"ord-3","Quantity":1}}
{"seq":6,"type":"StockReleased","data":{"ProductID":"prod-1","OrderID":"ord-3","Quantity":1}}
```

Any other folder would show only `stock: 5`, with no way of knowing that any of this happened.

**The log alone rebuilds everything.** With `EVENT_FILE` set, the process was stopped and started again. Real output:

```text
12-event-sourcing on port 8080, 8 events replayed
```

Products, customers, orders, and even the rule that emails are unique came back from the log. `scripts/verify.sh` runs the
contract suite in this mode too.

**Time travel.** Replaying only part of the history gives the state at that moment:

```go
if got := ReplayProduct(history[:i+1]).Stock; got != stock {
```

**Read models are disposable.** Delete a view and replay the log, and it is back. A view invented next year will
know the whole past:

```go
// Subscribe first replays the whole past, then delivers what comes next. That is how a brand new
// read model can be created at any time: it just reads the log from the beginning.
```

## The costs

**Replaying gets slower as history grows.** The benchmark orders one product thousands of times, so its stream keeps
growing, and each order replays all of it. Measured p50 was about 123 µs, against about 80 µs for the other single-process
folders, and it grows with the stream. The usual fix is a **snapshot**: save the state every N events, and replay only
what came after. It is not implemented here.

**Events are forever.** You cannot fix a wrong event, only add a correction. And every event type ever written must stay
readable, so changing a field means versioning it:

```go
// Decode turns a stored event back into its type. Old events live forever, so every type ever
// written must stay readable: this is the versioning problem of event sourcing.
```

**Some questions need a projection.** "Is this email taken?" cannot be answered by replaying one stream, so the command
side keeps a small lookup of its own:

```go
emails map[string]bool // a lookup that replaying one stream cannot answer
```

**Deleting data is hard.** Privacy laws may require erasing personal data, which collides directly with an immutable log.
Real systems encrypt personal data per person and throw away the key, or keep it outside the log.

## CQRS and Event Sourcing

They often appear together, and this folder uses both: commands append, and projections answer queries. But they
are independent. [Folder 10](../10-cqrs/README.md) has CQRS without Event Sourcing: there, the events update the views
and are thrown away. Here, the events are the only thing that is kept.

## How it is tested

Tests read as given, when, then. And the central promise has its own test:

```go
// The central promise: delete everything except the log, and all the state comes back.
func TestRestartingRebuildsEverythingFromTheLog(t *testing.T) {
```

## The gain and the price

| You gain | You pay |
|---|---|
| A complete audit trail, for free | Replay costs, and snapshots to manage |
| Answers to questions nobody thought of yet | Events that can never be changed |
| New read models that know the whole past | Versioning every event type, forever |
| Debugging by replaying what happened | Erasing personal data becomes hard |

## When to use it

- **Good for:** domains where history is the point: accounting, payments, audits, version control. Git itself works very much like this.
- **Avoid when:** only the current state matters, which is most systems. It is one of the hardest architectures to remove once adopted. Apply it only to the part of the system that really needs it.

## How to run

```bash
go run .                                    # in memory
EVENT_FILE=/tmp/events.jsonl go run .       # stop it, start it again, and everything is back
curl localhost:8080/orders/ord-3/history
go test ./...
```

**Measurement:** 10 files, 786 lines of code and 8 packages. See the [full comparison](../RESULTS.md).

**Next step:** [13. Serverless](../13-serverless/README.md).
