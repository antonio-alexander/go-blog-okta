## ----------------------------------------------------------------------
## This makefile can be used to execute common functions to interact with
## the source code, these functions ease local development and can also be
## used in CI/CD pipelines.
## ----------------------------------------------------------------------

env_file=.okta.env

# REFERENCE: https://stackoverflow.com/questions/16931770/makefile4-missing-separator-stop
help: ## Show this help.
	@sed -ne '/@sed/!s/## //p' $(MAKEFILE_LIST)

check-lint: ## validate/install golangci-lint installation
	which golangci-lint || (go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2)

lint: check-lint ## lint the source with verbose output
	@golangci-lint run --verbose

check-godoc: ## validate/install godoc
	@which godoc || (go install golang.org/x/tools/cmd/godoc@v0.1.10)

serve-godoc: check-godoc ## serve (web) the godocs
	@godoc -http :8080

build: ## build the source (latest)
	@docker compose build --build-arg GIT_COMMIT=`git rev-parse HEAD` --build-arg GIT_BRANCH=`git rev-parse --abbrev-ref HEAD`
	@docker image prune -f

run: ## run the service and its dependencies (docker) detached
	@docker compose --env-file ${env_file} up -d

stop:
	@docker compose --env-file ${env_file} down

clean: stop ## stop and clean docker resources
	@docker compose --env-file ${env_file} rm -f

okta-envs: ## loads okta envs
	@. ./.okta.env