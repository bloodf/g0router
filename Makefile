BINARY := g0router
CMD := ./cmd/g0router
UI_DIR := ui

.PHONY: build test lint vet ui ui-deps ui-test ui-e2e e2e e2e-api e2e-binary verify docker install clean

build: ui
	go build -o $(BINARY) $(CMD)

test: ui
	go test ./... -count=1

lint:
	go vet ./...

vet: lint

ui-deps:
	npm ci --prefix $(UI_DIR) --include=dev

ui: ui-deps
	npm run build --prefix $(UI_DIR)

ui-test: ui-deps
	npm --prefix $(UI_DIR) test -- --run

ui-e2e: ui-deps
	npm run e2e --prefix $(UI_DIR)

# Go API comprehensive e2e tests (34+ endpoint tests).
e2e-api:
	go test -run TestE2E -count=1 -v .

# Full e2e suite: Go API tests + UI build + Playwright UI tests.
e2e: build e2e-api ui-e2e

# Opt-in real-binary smoke test: builds the binary, runs it, exercises the HTTP
# surface (health, embedded UI, auth, /v1/models, control-plane routes).
e2e-binary:
	go test -tags e2ebin -run TestE2EBinary -count=1 .

verify: ui-deps
	go test ./... -count=1
	go vet ./...
	go build ./cmd/g0router
	npm --prefix $(UI_DIR) test -- --run
	npm run build --prefix $(UI_DIR)
	npm run e2e --prefix $(UI_DIR)
	$(MAKE) build
	git diff --check

docker:
	docker build -t g0router:latest .

install: build
	go install $(CMD)

clean:
	rm -f $(BINARY)
	rm -rf $(UI_DIR)/dist
