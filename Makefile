.PHONY: build dev frontend clean install test kill-stale-tests

BUILD_DIR := cmd/kanban
BINARY := player2-kanban

build: frontend
	go build -o $(BINARY) ./$(BUILD_DIR)

dev:
	go run ./$(BUILD_DIR) start --foreground

frontend:
	cd web && npm install && npm run build
	mkdir -p $(BUILD_DIR)/web/dist
	cp -r web/dist/* $(BUILD_DIR)/web/dist/

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)/web
	rm -rf web/dist web/node_modules

install: build
	cp $(BINARY) /usr/local/bin/

dev-frontend:
	cd web && npm run dev

kill-stale-tests:
	@go run scripts/kill-stale-tests.go 2>/dev/null || true

test: kill-stale-tests
	go test ./... ; TEST_EXIT=$$? ; go run scripts/kill-stale-tests.go 2>/dev/null || true ; exit $$TEST_EXIT
