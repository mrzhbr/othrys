.PHONY: build build-server build-cli test vet clean \
       server-up server-down server-build \
       client-install plugin-sync

BINARY_SERVER=bin/othrys-server
BINARY_CLI=bin/othrys

# ── Build ────────────────────────────────────────────────────────────
build: build-server build-cli

build-server:
	@mkdir -p bin
	go build -o $(BINARY_SERVER) ./cmd/server

build-cli:
	@mkdir -p bin
	go build -o $(BINARY_CLI) ./cmd/cli

# ── Server (Docker) ─────────────────────────────────────────────────
server-build:
	docker compose -f server/docker-compose.yml build

server-up:
	docker compose -f server/docker-compose.yml up -d

server-down:
	docker compose -f server/docker-compose.yml down

# ── Client Install ───────────────────────────────────────────────────
client-install:
	bash client/install.sh

# ── Dev ──────────────────────────────────────────────────────────────
test:
	go test ./... -v -count=1

vet:
	go vet ./...

# ── Plugin Sync ─────────────────────────────────────────────────────
plugin-sync:
	@echo "Syncing claude-plugin to marketplace and cache..."
	rsync -av --delete --exclude=.DS_Store --exclude=.claude claude-plugin/ ~/.claude/plugins/marketplaces/othrys-marketplace/plugins/othrys/
	rsync -av --delete --exclude=.DS_Store --exclude=.claude claude-plugin/ ~/.claude/plugins/cache/othrys-marketplace/othrys/1.0.0/
	@echo "Done. Restart Claude Code to pick up changes."

clean:
	rm -rf bin/
