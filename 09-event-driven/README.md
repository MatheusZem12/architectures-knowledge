# 09. Event-driven architecture

> Modules do not call each other. They **publish what happened**, and whoever cares reacts.

In every other folder, someone gives orders: `orders` calls `catalog` and waits. Here, nobody calls anybody. A module
says "stock was reserved", and has no idea who is listening.

## Structure

```text
09-event-driven/
├── main.go
├── architecture_test.go       ← no module may import another
├── events/events.go           ← every event: the system's flow, top to bottom
├── platform/
│   ├── broker/                ← in-memory stand-in for Kafka or RabbitMQ
│   └── web/
└── modules/
    ├── catalog/
    ├── customers/
    ├── orders/
    ├── payments/
    └── notifications/
```

## The flow of an order

No module knows this whole diagram. It emerges from each one reacting to what it cares about. This is called
**choreography**: there is no conductor.

```text
  HTTP ─► OrderRequested
              │ customers: does the customer exist?
              ▼
          CustomerConfirmed ────────────► OrderRejected (no customer)
              │ catalog: reserve the stock
              ▼
          StockReserved ────────────────► OrderRejected (no stock)
              │ orders: calculate the price
              ├─► OrderPlaced ─► notifications
              └─► ReservationRejected (bad coupon)
                      │ catalog: give the stock back
                      ▼
                  OrderRejected
```

## See it running

Run with `LOG_EVENTS=1` to watch the events. This is the real output for an order that succeeded:

```text
events.OrderRequested {RequestID:req-1 CustomerID:cust-1 Lines:[{ProductID:prod-1 Quantity:1}] Coupon:}
events.CustomerConfirmed {RequestID:req-1 CustomerID:cust-1 Lines:[{ProductID:prod-1 Quantity:1}] Coupon:}
events.StockReserved {RequestID:req-1 CustomerID:cust-1 Items:[{ProductID:prod-1 Quantity:1 UnitPriceCents:30000}] Coupon:}
events.OrderPlaced {OrderID:ord-1 CustomerID:cust-1}
```

And for one with an invalid coupon, where the compensation shows up as an event of its own:

```text
events.OrderRequested
events.CustomerConfirmed
events.StockReserved
events.ReservationRejected
events.OrderRejected
```

## What it looks like in practice

**A module only reacts.** Compare with the catalog in folder 07, which waited for HTTP calls:

```go
func (m *Module) handle(event any) error {
	switch e := event.(type) {
	case events.CustomerConfirmed:
		m.reserve(e)
```

**Consumers are free to come and go.** Nothing publishes "to notifications". It simply listens:

```go
case events.OrderPaid:
	m.add(e.CustomerID, e.OrderID, "order_paid")
```

Deleting that module changes nothing else. A new consumer, such as an audit log, plugs in the same way.

**The rule is a test.** It breaks the build if a module imports another:

```go
t.Errorf("%s must not import %s: publish an event instead", path, imported)
```

## The problem it solves from folder 07

In folder 07, when `notifications` went down, the notice was lost after three tries. Here, each subscriber has its own
queue, and a failed delivery is retried:

```go
// This is what folder 07 lacked: the consumer is down for a moment, and the event still arrives.
func TestBriefOutageDoesNotLoseTheEvent(t *testing.T) {
```

What fails every attempt is not thrown away either: it goes to a **dead letter** list, to be inspected and replayed.

```go
// DeadLetters lists what failed every attempt. It is kept, not lost: someone can inspect and replay it.
```

A slow consumer also holds nobody back, which `TestSlowSubscriberHoldsNobodyBack` proves.

**An honest limit:** this broker lives in memory. If the process dies, the queues die with it. Real durability
requires a real broker, and usually an outbox table as well. This folder shows the structure, not the guarantee.

## The new problems it creates

**HTTP wants an answer now, and events answer later.** The contract requires a synchronous response, so the module publishes
the first event and waits for the outcome, linked by a `RequestID`:

```go
// ask bridges the two worlds: HTTP wants an answer now, and events only give one later.
```

It is a common pattern, and it exists only to bring back what the architecture took away.

**State appears between events.** Between `ChargeRequested` and `PaymentApproved`, an order is in limbo. A cancellation
arriving at that moment would create a cancelled order that was charged. That is why there is a state that no other
folder needed:

```go
busy    map[string]bool // orders in the middle of a pay or cancel chain
```

**The flow has no single place to read.** In folder 07, the saga is one function. Here, it is spread across modules. If
something goes wrong, you need the event log to understand what happened.

**It has a measured cost.** Each event goes through a queue and a goroutine switch. In every run, the p50 of
`POST /orders` stayed about 20% above the other single-process folders: 94 to 96 µs against about 80 µs in the latest
runs. It is small next to the 487 µs of folder 07, and it would be larger with a real broker over the network.

## Choreography and orchestration

| | Choreography (this folder) | Orchestration (the saga in folder 07) |
|---|---|---|
| Who decides the next step | Each module, by reacting | One central coordinator |
| Where to read the flow | Spread out, in the events | In one function |
| Adding a consumer | Changes nothing else | Changes the coordinator |
| Risk | Nobody sees the whole flow | The coordinator knows too much |

## How it is tested

A test speaks the system's language: given these events, expect those events. A small subscriber plays the part of the
other modules:

```go
case events.ReservationRejected:
	compensated.Add(1)
```

## The gain and the price

| You gain | You pay |
|---|---|
| Publishers do not know their consumers | The flow is harder to follow and debug |
| New consumers without changing anything | Intermediate states to handle |
| Brief outages do not lose events | Eventual consistency, and a bridge when HTTP needs an answer |
| A slow consumer holds nobody back | Duplicates and ordering to think about with a real broker |

## When to use it

- **Good for:** several independent reactions to the same fact, integration between teams or systems, and work that can be done later.
- **Avoid when:** the flow is simple and the user is waiting for the answer. Here, the contract forced a synchronous bridge, which is a sign that this store is not the ideal case for it.

It also combines with the others: events usually connect the modules of a [modular monolith](../04-modular-monolith/README.md) or
the services in [folder 07](../07-microservices/README.md).

## How to run

```bash
LOG_EVENTS=1 go run .     # shows each event in the log
go test ./...
```

**Measurement:** 13 files, 807 lines of code and 9 packages. See the [full comparison](../RESULTS.md).

**Next step:** [10. CQRS](../10-cqrs/README.md).
