# 07. Microservices

> Each part of the system is a **separate process**, with its own data, and communicates only over the network.

Until now, every folder compiled to a single binary. This one compiles to six. The business rules are the same: what
is new is that **now everything can fail halfway**.

## Structure

```text
07-microservices/
├── build.sh  run.sh  test.sh
└── services/
    ├── gateway/         ← entry point (8080): only routes
    ├── catalog/         ← owner of products and stock (8081)
    ├── customers/       ← owner of customers (8082)
    ├── orders/          ← coordinates the sale (8083)
    │   ├── main.go      ← the saga
    │   ├── clients.go   ← network calls to the other services
    │   └── pricing.go   ← pricing rules
    ├── payments/        ← simulates the payment gateway (8084)
    └── notifications/   ← stores the notices (8085)
```

Each service has its **own `go.mod`**, and none imports another's code. Even `writeJSON` is duplicated, on purpose: a
shared library would tie the deploys back together. The contract between them is HTTP alone.

## How the parts talk

```text
  client ──► gateway ──┬──► catalog
                       ├──► customers
                       ├──► notifications
                       └──► orders ──┬──► customers       (does the customer exist?)
                                     ├──► catalog         (reserve and release)
                                     ├──► payments        (charge)
                                     └──► notifications   (in the background)
```

## Path of a `POST /orders`

1. `gateway` forwards to `orders` (**network hop 1**).
2. `orders` asks `customers` whether the customer exists (**hop 2**).
3. `orders` requests the reservation from `catalog` (**hop 3**).
4. `orders` calculates the price and, if the coupon fails, calls `catalog` to undo the reservation (**extra hop**).
5. `orders` saves the order and responds.
6. In the background, `orders` tells `notifications`.

## What really changes: the saga

In a single database, a transaction would guarantee all or nothing. Across services **there is no shared transaction**.
Instead, there is the **saga**: a sequence of steps, each with an undo step.

```go
// R08: a refused coupon triggers the compensation, undoing step 1
value, couponErr := discount(in.Coupon, order.SubtotalCents, !s.hasActiveOrder(in.CustomerID))
if couponErr != nil {
	if err := s.clients.Release(in.Items); err != nil {
		// This is the worst case of distributed systems: the stock stays stuck until
		// someone reconciles it. A real system would use an outbox and retries.
		log.Printf("COMPENSATION FAILED, stock stuck: %v", err)
	}
	return Order{}, couponErr
}
```

That comment is the heart of the matter. What if `Release` fails too? The stock ends up wrong, and no line of code
alone solves that. Patterns exist (outbox, reconciliation, idempotency), but they all take work.

**A function call became a network call**, and each one brings new questions. Was the service slow? Did it fail before
or after executing? Can it be retried without charging twice?

```go
http:          &http.Client{Timeout: 2 * time.Second},
```

## What happens when something goes down

This is the big argument for the architecture, and it also shows the price. **Real result**, killing processes while
the system runs:

| Situation | `POST /orders` | Get an existing order | Side effect |
|---|---|---|---|
| All up | 201 | 200 | |
| `notifications` down | **201**, the sale goes on | 200 | `GET notifications` returns 503, and **the notice is lost** |
| `catalog` down | **503** | **200**, still answers | Nothing is sold, but the rest works |

In a monolith, a memory leak or a `panic` in a goroutine takes everything down. (`net/http` recovers `panic`s inside a
handler, but not those of other goroutines.) Here, the outage is **partial**. But look at the cost in the log actually
produced by the test:

```text
notification lost: order_created of order ord-2
```

The code tries three times and gives up. The order was created, and the customer never got the notice. With
`order_paid`, it would be a charge with no confirmation. Solving this requires a broker and an outbox table, which are
not here.

```go
// Notify runs in the background, with retries: a notification outage must not
// take down a sale. The price is possibly losing the notice if every attempt fails.
```

## The measured cost

Same rules, same machine, **on localhost, with no real network latency** (see [`RESULTS.md`](../RESULTS.md)):

| | Monoliths (02 to 06) | Microservices |
|---|---|---|
| p50 of `POST /orders` | about 83 µs | about 500 µs, **about 6 times** |
| Orders per second, sequentially | about 11,000 | about 1,900 |
| Memory | about 18 MB | about 108 MB, **about 6 times** |
| Binaries | 9.5 MB | 58 MB, in 6 processes |

Across all runs, the latency ratio stayed between 4.3 and 6 times.

On a real network, each hop adds even more time. On the other hand, notice what this table does **not** measure: the
ability to scale only the catalog, or for a team to deploy without talking to the others. Those are the real gains,
and they are organizational.

Note also that the 708 lines of code are misleading. They came in below hexagonal because each service here is flat
inside. Also left out is what a real system would need: distributed tracing, service discovery, a broker, an outbox,
circuit breakers, a pipeline per service and Kubernetes. Most of the real cost of this architecture does not appear in
the code, but in operations.

## How it is tested

Testing the saga requires simulating the other services as HTTP servers:

```go
remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
```

Each service's tests pass in isolation. **Only the contract suite, with everything up, proves the whole works.**
Integration tests become expensive and indispensable.

## The gain and the price

| You gain | You pay |
|---|---|
| Independent deploys per team | 4 to 6 times the latency, and more on a real network |
| Independent scaling per service | No transactions: eventual consistency and sagas |
| Partial outage, instead of total | Operations: monitoring, tracing, deploying N services |
| Freedom of technology per service | Debugging a request that crosses 4 processes |
| A physical boundary, impossible to bypass | API contracts to version between teams |

## When to use it

Microservices mainly solve an **organizational problem**: many teams blocking each other in the same code.

- **Good for:** many teams (dozens of people), parts with very different scaling needs and teams mature in operations.
- **Avoid when:** the team is small, the domain still changes a lot (getting the boundaries wrong here is expensive) or there is no operations structure.

The safest path is usually to start with a [modular monolith](../04-modular-monolith/README.md) and extract a service
when there is a concrete reason. Compare the two: `orders` does almost the same thing in both folders, with
`m.catalog.Reserve` there and `s.clients.Reserve` here.

An antipattern to avoid is the **distributed monolith**: services that must start together and call each other in
chains. You pay the cost of both architectures and get the benefit of neither. This example carries a bit of that risk,
since `orders` depends synchronously on `catalog` and `customers`.

## How to run

```bash
./run.sh                       # builds and starts the 6 processes; gateway on port 8080
BASE_PORT=9000 ./run.sh        # uses other ports
./test.sh                      # each service's tests
```

To see the partial failure yourself, kill a service in another terminal and repeat the `curl` commands from the
[main README](../README.md):

```bash
pkill -x notifications     # the sale goes on, and the notice is lost
pkill -x catalog           # creating an order returns 503, and getting one still works
```

**Next step:** [08. MVC](../08-mvc/README.md).
