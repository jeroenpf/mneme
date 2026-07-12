# Mneme dev orchestration. See .architecture/specs/2026-05-26-vue-foundation-design.md.
#
# Run `make help` (or bare `make`) to list targets.

COMPOSE := docker compose
WEB     := web
GO_PKGS := ./...

.DEFAULT_GOAL := help
.PHONY: help dev up down logs psql seed test build tidy clean reset setup-host verify-host

help: ## Show this list
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

dev: up ## Start full stack: postgres + Go (via air) + Vite. Ctrl-C stops Vite; backend stays running.
	cd $(WEB) && npm install --silent && \
		NODE_EXTRA_CA_CERTS="$$(mkcert -CAROOT)/rootCA.pem" npm run dev

up: ## Bring up backend containers in the background
	$(COMPOSE) up -d

down: ## Stop containers. Volumes preserved.
	$(COMPOSE) down

logs: ## Tail backend logs (app + postgres)
	$(COMPOSE) logs -f --tail=100 app postgres

psql: ## Open a psql shell inside the postgres container
	$(COMPOSE) exec postgres psql -U mneme -d mneme

seed: ## Load dev sample data (scripts/dev-seed.sql) into postgres
	$(COMPOSE) exec -T postgres psql -U mneme -d mneme -v ON_ERROR_STOP=1 < scripts/dev-seed.sql

test: ## Run Go and Vue tests
	go test $(GO_PKGS)
	cd $(WEB) && npm test

build: ## Build the Vue app, copy it into internal/web/dist for embedding, then build the Go binary
	cd $(WEB) && npm install --silent && npm run build
	rm -rf internal/web/dist
	cp -r $(WEB)/dist internal/web/dist
	touch internal/web/dist/.gitkeep
	go build -o cmd/server/server ./cmd/server

tidy: ## Refresh dependency manifests on both sides
	go mod tidy
	cd $(WEB) && npm install

clean: ## Remove build artefacts (keeps node_modules and Go caches)
	rm -rf $(WEB)/dist $(WEB)/.vite internal/web/dist
	mkdir -p internal/web/dist && touch internal/web/dist/.gitkeep

reset: ## DESTRUCTIVE: drop containers AND postgres volume. Requires RESET=yes.
ifeq ($(RESET),yes)
	$(COMPOSE) down -v
else
	@echo "This will destroy the postgres volume."
	@echo "Re-run with: RESET=yes make reset"
	@exit 1
endif

setup-host: ## One-time host setup: mkcert CA + cert, LaunchDaemon, lo0 alias, /etc/hosts. Run with sudo.
	@bash scripts/setup-host.sh

verify-host: ## Scorecard for the mneme.dev host setup (non-destructive, no sudo)
	@bash scripts/verify-host.sh
