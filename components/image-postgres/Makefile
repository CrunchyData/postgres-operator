BUILDAH ?= buildah
BUILDAH_BUILD ?= $(BUILDAH) build

POSTGRES_IMAGE ?= localhost/crunchy-postgres:latest
POSTGIS_IMAGE ?= localhost/crunchy-postgres-gis:latest
UPGRADE_IMAGE ?= localhost/crunchy-upgrade:latest

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

.PHONY: all build-postgres build-postgis build-postgres-upgrade
all: build-postgres build-postgis build-postgres-upgrade

build-postgres: ## Build the Postgres image
	$(BUILDAH_BUILD) \
		--tag $(POSTGRES_IMAGE) \
		-f Dockerfile.postgres

build-postgis: ## Build the PostGIS image (Requires the Postgres image)
build-postgis: build-postgres
	$(BUILDAH_BUILD) \
		--tag $(POSTGIS_IMAGE) \
		-f Dockerfile.postgis

build-postgres-upgrade: ## Build the Postgres upgrade image (Requires the Postgres image)
build-postgres-upgrade: build-postgres
	$(BUILDAH_BUILD) \
		--from $(POSTGRES_IMAGE) \
		--tag  $(UPGRADE_IMAGE) \
		-f Dockerfile.postgres-upgrade
