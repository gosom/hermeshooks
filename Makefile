.PHONY: default
default: help

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: server
server: ## server starts the webserver
	go run cmd/hermeshooks/main.go server

.PHONY: worker
worker: ## worker starts a worker
	go run cmd/hermeshooks/main.go worker

.PHONY: fixtures
fixtures: ## fixtures inserts some dummy jobs
	go run cmd/hermeshooks/main.go fixtures

.PHONY: migrate
migrate: ## migrate 
	tern migrate --config migrations/tern.conf --migrations ./migrations

.PHONY: migrate-down
migrate-down: ## migrate-down 
	tern migrate --destination -1 --config migrations/tern.conf --migrations ./migrations

.PHONY: migrate-create
migrate-create: ## migrate-create 
	tern new $(NAME) -m ./migrations

