# The mise.toml in this folder pins the Go version. If needed, run `mise install`.
.PHONY: verify metrics principles rules

verify:        ## builds, tests and runs the contract on all architectures
	./scripts/verify.sh

metrics:       ## measures everything and rewrites RESULTS.md
	./scripts/metrics.sh

principles:    ## tests the SOLID and Clean Code examples
	cd principles && go test ./...

rules:         ## checks that the rules are tagged in all architectures
	./scripts/where-rule.sh --check
