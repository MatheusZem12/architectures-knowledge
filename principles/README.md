# Code principles

**SOLID and Clean Code are not architectures.** Architecture answers "how is the system divided?". Principles answer
"how should this piece of code be written?". That is why it would make no sense to implement the whole store "in
SOLID": they apply inside any of the folders `01` to `07`.

Here, each principle is shown **before and after**, using the rules of the same store, with a test proving the behavior
did not change.

| Subject | What it is | Where |
|---|---|---|
| Clean Code | How to write so that someone else understands | [`clean-code/`](clean-code/README.md) |
| SOLID | Five principles on dividing responsibilities and dependencies | [`solid/`](solid/README.md) |

```bash
go test ./...      # inside this folder
```
