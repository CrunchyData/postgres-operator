# https://www.gnu.org/software/make/manual/make.html#index-secondary-targets
.SECONDARY:

BUILDAH ?= buildah
BUILDAH_BUILD ?= $(BUILDAH) build

PGBACKREST_IMAGE ?= localhost/crunchy-pgbackrest:latest

##@ General

.PHONY: help
help: ALIGN=18
help: ## Display this help.
	@awk ' \
		BEGIN { FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n" } \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-$(ALIGN)s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
	' $(MAKEFILE_LIST)


##@ Build

.PHONY: build
build: ## Build the pgBackRest image
build: Dockerfile
	$(BUILDAH_BUILD) \
		--tag $(PGBACKREST_IMAGE) \
		--file $< --format 'docker' --layers ../..
