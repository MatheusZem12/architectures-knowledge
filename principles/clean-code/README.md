# Clean Code

> Code is read far more often than it is written. Write for whoever reads it.

The two versions below calculate **exactly the same thing**. The file
[`same_behavior_test.go`](same_behavior_test.go) checks this across 192 combinations of price, quantity, coupon and
customer.

## Before ([`before.go`](before.go))

```go
func calc(d []map[string]int, c string, f bool) (int, int, int, int) {
	t := 0
	for _, x := range d {
		t += x["p"] * x["q"]
	}
	r := 0
	if c == "WELCOME10" && f {
		r = t * 10 / 100
	} else if c == "BIG20" && t >= 50000 {
		r = t * 20 / 100
	}
	s := 2000
	if t-r >= 20000 {
		s = 0
	}
	return t, r, s, t - r + s
}
```

## After ([`after.go`](after.go))

```go
func CalculateTotals(items []Item, coupon string, isFirstOrder bool) Totals {
	subtotal := subtotalOf(items)
	discount := discountFor(coupon, subtotal, isFirstOrder)
	shipping := shippingFor(subtotal - discount)
	return Totals{subtotal, discount, shipping, subtotal - discount + shipping}
}
```

This function now reads like the business rule. If you want the shipping details, you open `shippingFor`. If you do
not, you do not have to.

## What changed, and why

| Practice | Before | After | Why it matters |
|---|---|---|---|
| Names that reveal intent | `t`, `r`, `s`, `f` | `subtotal`, `discount`, `shipping`, `isFirstOrder` | Nobody has to decipher them |
| No magic numbers | `50000`, `20000`, `2000` | `bigOrderMinimumCents` | The number gains meaning, and changes in one place |
| Small functions | One does everything | Each does one thing | You can read just the part you care about |
| Types instead of maps | `map[string]int` with `"p"` and `"q"` | `Item{UnitPriceCents, Quantity}` | The compiler finds the typo |
| Named return | `(int, int, int, int)` | `Totals` | The order cannot be swapped by mistake |
| Explained boolean parameter | `f bool` | `isFirstOrder bool` | The signature explains itself |
| Comment about the why | None | "the store never gives a discount in fractions of a cent" | Says what the code cannot |

## Comments: the why, never the what

```go
// percentOf rounds down, because the store never gives a discount in fractions of a cent.
func percentOf(value, percent int) int {
```

A comment like `// sums the subtotal` above a loop just repeats the code and ages badly. Prefer a good name. Comment
what the code does not say: a decision, a limit, a reason.

## Other practices worth remembering

- **Exit early.** Validate and return at the start, instead of nesting `if`s. Compare `Place` in folder 02 with the nesting in folder 01.
- **One level of abstraction per function.** Do not mix business rules with writing JSON in the same function.
- **Leave it cleaner than you found it.** Small frequent improvements beat one big rewrite.
- **Test before refactoring.** Without a test like `same_behavior_test.go`, it is not refactoring, it is hoping.

## Without overdoing it

Clean Code is a means, not the goal. Rules followed to the letter also do harm: breaking a clear ten-line function into
five two-line ones forces you to jump between them to understand a simple idea. And three similar lines are often
better than a premature abstraction. The question is always: **did it get easier for the next person?**
