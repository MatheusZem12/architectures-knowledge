# architectures-knowledge

The **same store**, implemented in 10 different ways, in Go. The goal is to see what each architecture looks like in
code, not just to read its definition.

All implementations have the same endpoints and the same [14 business rules](SPEC.md). This is proven by a
[contract suite](contract/contract_test.go): 25 tests that speak only HTTP and know nothing about the code behind it.
Only the structure changes from one folder to the next, so **any difference you notice comes from the architecture**.

## Where to start

Read in order. Up to folder 07, each one solves a problem the previous one left behind. Folders 08 to 10 are independent and can be read in any order, though 09 builds on 07. If MVC is what you already know from web frameworks, it makes a good first stop.

| # | Folder | The idea | The problem it solves |
|---|---|---|---|
| 01 | [Spaghetti](01-spaghetti/README.md) | Everything in one file | None: it is the baseline |
| 02 | [Layered](02-layered/README.md) | Split by type of work | Rules mixed with HTTP and data |
| 03 | [Package by feature](03-package-by-feature/README.md) | Split by subject | A change scattered across many folders |
| 04 | [Modular monolith](04-modular-monolith/README.md) | Modules with enforced boundaries | Subjects coupled without anyone noticing |
| 05 | [Hexagonal](05-hexagonal/README.md) | Business at the center, technology plugged in | Business depending on the database |
| 06 | [Clean Architecture](06-clean-architecture/README.md) | Rings, and one use case per file | Same as 05, with more rigor |
| 07 | [Microservices](07-microservices/README.md) | Separate processes, connected by the network | Teams and scaling getting in each other's way |
| 08 | [MVC](08-mvc/README.md) | Model, view and controller | Rules mixed with how things are shown |
| 09 | [Event-driven](09-event-driven/README.md) | Modules react to facts, and never call each other | Modules that must know who comes next |
| 10 | [CQRS](10-cqrs/README.md) | One model to decide, another to read | One model doing two very different jobs |
| | [SOLID](principles/solid/README.md) | Five principles, before and after | Code that is hard to extend and test |
| | [Clean Code](principles/clean-code/README.md) | Writing for whoever reads it next | Code that is hard to understand |

Other common names: folder 02 is also called a *monolith by responsibility*, and folder 03 a *monolith organized by module*.

## Not all of this is "architecture"

These terms often appear in the same list, but they answer **different questions**. Separating them may be the most
useful lesson in this repository:

| Question | Possible answers | Folders |
|---|---|---|
| **How many pieces go to production?** | Monolith or microservices | all the others, versus 07 |
| **How is the code divided inside?** | Layers, features, modules, hexagonal, clean | 02 to 06 |
| **How is the interface separated from the rules?** | MVC | 08 |
| **How do the parts talk?** | Direct calls or events | 09, against the others |
| **How is data modeled?** | One model, or separate models to write and read | 10, against the others |
| **How is each piece of code written?** | SOLID, Clean Code | `principles/` |

Since these axes are independent, **they combine**:

- A modular monolith with each module hexagonal is a common and very good combination.
- A microservice can have layers inside.
- Event-driven usually connects services or modules, and CQRS is often applied to just one of them.
- You can write clean code in any of them, and horrible code in hexagonal too.

"Monolith or hexagonal?" is a false choice, because folder 05 **is** a monolith.

Two names that are often confused:

- **Package by feature (03) and modular monolith (04):** the folders look the same. The difference is whether the boundary is just a convention or is enforced by the compiler.
- **Clean Code and Clean Architecture:** not the same thing. One is about writing, the other about structure.

## The result

These numbers are measured by [`scripts/metrics.sh`](scripts/metrics.sh). The full table is in [`RESULTS.md`](RESULTS.md).

| Architecture | .go files | Lines of code | Packages | Processes | Memory | p50 `POST /orders` |
|---|---:|---:|---:|---:|---:|---:|
| 01 Spaghetti | 1 | 279 | 1 | 1 | 24 MB | 92 µs |
| 02 Layered | 14 | 592 | 6 | 1 | 17 MB | 83 µs |
| 03 By feature | 8 | 444 | 7 | 1 | 19 MB | 82 µs |
| 04 Modular | 15 | 678 | 13 | 1 | 17 MB | 83 µs |
| 05 Hexagonal | 16 | 841 | 8 | 1 | 17 MB | 84 µs |
| 06 Clean | 25 | 891 | 7 | 1 | 19 MB | 83 µs |
| 07 Microservices | 9 | 708 | 6 | 6 | 108 MB | 500 µs |
| 08 MVC | 14 | 557 | 4 | 1 | 20 MB | 86 µs |
| 09 Event-driven | 13 | 807 | 9 | 1 | 17 MB | 100 µs |
| 10 CQRS | 10 | 605 | 5 | 1 | 19 MB | 83 µs |

Absolute numbers change between runs, depending on what else the machine is doing: earlier runs had the monoliths at 114
to 146 µs. The comparison between folders in the same run is what means something.

What you can conclude:

1. **Code layers have no noticeable performance cost.** Folders 02, 03, 04, 05, 06 and 10 stay within a few microseconds of each other, and their order changes between runs. "Too much abstraction makes it slow" does not hold here.
2. **The cost of crossing the network is real.** Microservices were 4.3 to 6 times slower across runs, and use about 6 times the memory, even on localhost.
3. **Queues have a smaller, but real, cost.** Event-driven stays about 20% above the other single-process folders, in every run.
4. **More structure means more code.** Clean architecture has more than three times the lines of spaghetti, for the same behavior.
5. **Spaghetti won nothing at runtime.** It is the smallest, but it was slower than folders 02 to 06 in every run.

What the numbers do **not** show: the cost of changing the system and the cost of operating it, which usually matter
far more. They also only apply to this example: in memory and on localhost, with no database. In a real system, the
database dominates latency, and the gaps between the monoliths would shrink further.

## The cost of each change

This table was assessed by reading the code, **not measured**. It shows what must change in each case.

| Change | 01 | 02 | 03 | 04 | 05 | 06 | 07 | 08 | 09 | 10 |
|---|---|---|---|---|---|---|---|---|---|---|
| **New coupon** | 1 `if` | 1 file | 1 file | 1 file | 1 type | 1 type | 1 file and 1 deploy | 1 file | 1 file | 1 file |
| **Test the coupon without starting anything** | Impossible | With real repositories | With real stores | Pure function | Pure function | Pure function | Pure function | Pure function | Pure function | Pure function |
| **Swap persistence** | Almost everything | 4 repositories | 4 folders, mixed with HTTP | 4 internal stores | **1 new adapter** | **1 new gateway** | 4 services, one at a time | The whole model | 4 modules | Each side on its own |
| **Swap the payment gateway** | Find the `if` | 1 file | 1 file | 1 module | **1 new adapter** | **1 new gateway** | **1 service** | 1 function | **1 module** | **1 type** |
| **Remove notifications** | Several spots | Service and repository | `orders` and `main` | **Just the folder and `main`** | Core and adapters | 3 use cases and more | `orders` and 1 service | Model and controller | **Just the folder and `main`** | 1 view and 1 route |
| **New JSON or HTML format** | Almost everything | Model and handlers | Each feature | Each module | **1 adapter** | **1 presenter** | Each service | **1 view** | Each module | **The views** |
| **Deploy only orders** | No | No | No | No | No | No | **Yes** | No | No | No |

Note the first row: **a well-contained change is cheap in any architecture**, even spaghetti. The differences show up
in changes that cut across the system.

## How to choose

There is no universal answer, but this is a good starting point:

```text
Is it disposable (a script or prototype)?
└─ yes → anything, even 01

Is it a small CRUD, with few rules?
└─ yes → 03 (or 02, if the team already knows it)

Will it grow and have important rules?
└─ yes → 04, using hexagonal (05) in the modules with the most rules

Many teams blocking each other, or parts with very different scaling needs?
└─ yes → 07, preferably extracted from a modular monolith

A web app with server-rendered pages?
└─ yes → 08 at the edge, adding 04 or 05 if the rules grow

Many independent reactions to the same fact?
└─ yes → 09, in the part of the system that needs it

Far more reads than writes, or screens that combine a lot of data?
└─ yes → 10, in the part of the system that needs it
```

The most expensive mistake is usually choosing by fashion: microservices on a team of three, or four rings on a CRUD.
The right architecture is the simplest one that solves the problems you **have**, not the ones you might have someday.

## Running it

The Go version is pinned by this folder's `mise.toml` (run `mise install` if needed). There are no external
dependencies: only the standard library.

```bash
./scripts/verify.sh                 # builds, tests and runs the contract on all of them
./scripts/verify.sh 05-hexagonal    # just one
./scripts/metrics.sh                # measures everything and rewrites RESULTS.md
./scripts/where-rule.sh R08         # where rule R08 lives in each architecture
cd principles && go test ./...      # SOLID and Clean Code tests
```

To use an architecture by hand:

```bash
cd 05-hexagonal && go run .         # or ./run.sh in folder 07

P=$(curl -s localhost:8080/products -d '{"name":"Keyboard","price_cents":30000,"stock":5}' | jq -r .id)
C=$(curl -s localhost:8080/customers -d '{"name":"Ana","email":"ana@example.com"}' | jq -r .id)
O=$(curl -s localhost:8080/orders -d "{\"customer_id\":\"$C\",\"coupon\":\"WELCOME10\",\"items\":[{\"product_id\":\"$P\",\"quantity\":1}]}" | tee /dev/stderr | jq -r .id)
curl -s -X POST localhost:8080/orders/$O/pay
curl -s localhost:8080/customers/$C/notifications
```

## A good comparison exercise

Each rule is tagged in the code with a comment. See where the coupon rule lives:

```text
$ ./scripts/where-rule.sh R08
== 01-spaghetti
main.go:185: // R08
== 02-layered
internal/service/order_service.go:116:// R08
== 03-package-by-feature
orders/orders.go:121:// R08
== 04-modular-monolith
modules/orders/internal/pricing/pricing.go:17:// Discount calculates the discount in cents (R08).
modules/orders/api.go:84: // R08: a refused coupon undoes the reservation (compensation)
== 05-hexagonal
internal/core/domain/coupon.go:14:// R08
== 06-clean-architecture
entities/coupon.go:12:// R08
== 07-microservices
services/orders/pricing.go:11:// discount applies R08.
services/orders/main.go:116: // R08: a refused coupon triggers the compensation, undoing step 1
== 08-mvc
models/order.go:37:// Discount is pure, so it can be tested with nothing around it (R08).
== 09-event-driven
modules/catalog/catalog.go:40: // R08: a refused coupon gives the stock back, and only then the rejection goes
modules/orders/pricing.go:11:// discount applies R08.
modules/orders/orders.go:92: // R08: the stock was already reserved, so the compensation has to go out
== 10-cqrs
write/handler.go:66: active    map[string]int // active orders per customer: answers R08 without scanning
write/pricing.go:9:// discountFor applies R08.
```

The file path alone says a lot: from `main.go` to `service`, then `orders`, then `domain`.

To really practice, **add a new rule to all the folders**: for example, a `FREESHIPPING` coupon or a limit of 3
pending orders per customer. Write the test in `contract/contract_test.go` first. Feeling where each architecture
helps and where it gets in the way teaches more than any text.

## What makes sure this is right

| Check | What it proves |
|---|---|
| Contract suite (25 tests) | The 10 architectures behave the same, including under concurrent orders |
| `architecture_test.go` (04, 05, 06, 08, 09 and 10) | The dependency rules hold, and the build breaks if someone violates them |
| `where-rule.sh --check` | All rules are tagged in all folders |
| Go's race detector, with the contract repeated 8 times (08, 09 and 10) | No data races, including in the asynchronous folder |
| Deliberately injected bugs | The tests or the compiler caught each rule and dependency error inserted into a copy of the project |

## What was left out

Authentication, a real database, pagination and observability were left out so they would not get in the way of the
comparison. Data stays in memory. The exception is folder 05's file adapter, which exists to prove persistence can be
swapped.

Some ideas that could become new folders:

- **Event Sourcing:** storing the facts, instead of the current state.
- **Vertical Slice:** each feature as its own slice, from the entry point down to the data.
- **Serverless:** functions run on demand, as in AWS Lambda.
- **SOA:** the predecessor of microservices, still common in large companies.
- **Microkernel:** a small core with plugins, as in VS Code.
- **DDD:** aggregates and bounded contexts, modeled seriously. It is more a way of thinking about the domain than a folder structure.

To add one, create `11-name/`, add the folder to `go.work` and implement [`SPEC.md`](SPEC.md). `verify.sh` and
`metrics.sh` will find it on their own.
