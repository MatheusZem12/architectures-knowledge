# SOLID

Five principles on dividing responsibilities and organizing dependencies. Each folder has a `before.go`, an `after.go`
and a test that really runs. See also the "Where it appears" column, which shows the principle in use in the
repository's architectures.

| Letter | Principle | In one sentence | Where it appears |
|---|---|---|---|
| **S** | Single Responsibility | One reason to change | `06-clean-architecture/usecases/`, one file per use case |
| **O** | Open/Closed | Extend without editing what already works | `05-hexagonal/internal/core/domain/coupon.go` |
| **L** | Liskov Substitution | Implementers keep the promise | `memory` and `jsonfile` in folder 05 pass the same contract |
| **I** | Interface Segregation | Small interfaces, sized to their use | `06-clean-architecture/usecases/gateways.go` |
| **D** | Dependency Inversion | The business defines the interface, and the detail implements it | `05-hexagonal/internal/core/ports/ports.go` |

## S: Single Responsibility ([`srp/`](srp/))

**It does not mean "do only one thing".** It means having **a single reason to change**, that is, answering to a single
person or business area.

```go
func (m *OrderManagerBefore) Process(priceCents, quantity int) string {
	total := priceCents * quantity
	receipt := fmt.Sprintf("Total: $%d.%02d", total/100, total%100)
	m.saved = append(m.saved, receipt)
	return receipt
}
```

This method changes if sales alters the price, if design alters the receipt, or if infrastructure alters the storage.
After the split, each change affects only one type: `Pricing`, `ReceiptFormatter` or `ReceiptLog`.

## O: Open/Closed ([`ocp/`](ocp/))

**Open for extension, closed for modification.** Before, each new coupon edited the `switch`. After, a new coupon is
just a new type:

```go
type Coupon interface {
	Discount(subtotal int) int
}
```

The test proves it: a coupon of a brand new type, defined in the test file itself, works without any line of
`after.go` changing.

```go
type fixedCoupon struct{ cents int }

func (c fixedCoupon) Discount(int) int { return c.cents }
```

**Careful:** do not create extension points in advance. With two coupons, a `switch` is great. Extracting the interface
pays off when the cases really start to grow.

## L: Liskov Substitution ([`lsp/`](lsp/))

**Whatever implements an interface must keep its promise.** Compiling is not enough:

```go
func (g *GiftCardBefore) Charge(cents int) { g.Balance += cents }
func (g *GiftCardBefore) Refund(int)       { panic("gift cards have no refund") }
```

The code compiles, but breaks in production when someone uses `GatewayBefore` trusting the contract. The solution is
for the interface to promise only what everyone delivers, separating the refund:

```go
// Cancel only accepts what can refund. Passing a GiftCard is now a compile error:
// the problem moved from production to the compiler.
func Cancel(gateway Refunder, cents int) { gateway.Refund(cents) }
```

**A real example in the repository:** `memory` and `jsonfile`, in folder 05, are truly substitutable. The proof is that
both pass the same 25 contract tests.

## I: Interface Segregation ([`isp/`](isp/))

**Nobody should depend on methods they do not use.** `RevenueBefore` only uses `All`, but demands a six-method
interface. The consequence shows up in the test:

```go
type fatFake struct{ orders []Order }

func (f fatFake) Add(Order)                 {}
func (f fatFake) Find(string) (Order, bool) { return Order{}, false }
func (f fatFake) All() []Order              { return f.orders }
func (f fatFake) Delete(string)             {}
func (f fatFake) Update(Order)              {}
func (f fatFake) Backup() error             { return nil }
```

With the interface sized to its use, one line does it:

```go
type slimFake []Order

func (f slimFake) All() []Order { return f }
```

In Go, this comes naturally: interfaces are satisfied implicitly, so the consumer can declare its own. As the Go
proverb says: *"the bigger the interface, the weaker the abstraction"*.

## D: Dependency Inversion ([`dip/`](dip/))

**The business rule must not depend on details.** Both must depend on an abstraction, and that abstraction belongs to
the business.

Before, the business creates the detail by itself, and there is no way to test it without sending email:

```go
func NewOrdersBefore() *OrdersBefore { return &OrdersBefore{mailer: &SMTPMailerBefore{}} }
```

After, it receives from outside anything that knows how to notify:

```go
func New(notifier Notifier) *Orders { return &Orders{notifier: notifier} }
```

This is the principle most tied to architecture: **hexagonal and clean architecture are, in essence, DIP applied to the
whole system.** And a point that often confuses people:

| Term | What it is |
|---|---|
| **Dependency inversion** | The principle: the business owns the abstraction |
| **Dependency injection** | The technique: receiving the dependency from outside, for example in the constructor |

## Without overdoing it

SOLID is a list of warning signs, not a to-do list. An interface with a single implementation, created "just in case",
is just one more file to read. Apply a principle when you feel the pain it solves.
