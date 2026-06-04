BINARY := g0router
CMD := ./cmd/g0router
UI_DIR := ui

.PHONY: build test lint vet ui docker install clean

build: ui
	go build -o $(BINARY) $(CMD)

test: ui
	go test ./... -count=1

lint:
	go vet ./...

vet: lint

ui:
	npm ci --prefix $(UI_DIR) --include=dev
	npm run build --prefix $(UI_DIR)

docker:
	docker build -t g0router:latest .

install: build
	go install $(CMD)

clean:
	rm -f $(BINARY)
	rm -rf $(UI_DIR)/dist
