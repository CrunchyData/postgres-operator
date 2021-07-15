#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

push_trap_exit() {
	local -a array
	eval "array=($(trap -p EXIT))"
	# shellcheck disable=SC2064
	trap "$1;${array[2]-}" EXIT
}

# Store anything in a single temporary directory that gets cleaned up.
TMPDIR=$(mktemp -d)
push_trap_exit "rm -rf '${TMPDIR}'"
export TMPDIR

validate_bundle_image() {
	local container="$1" directory="$2"
	directory=$(cd "${directory}" && pwd)

	cat > "${TMPDIR}/registry.config" <<-SSL
	[req]
	distinguished_name = req_distinguished_name
	x509_extensions = v3_ext
	prompt = no
	[req_distinguished_name]
	commonName = localhost
	[v3_ext]
	subjectAltName = @alt_names
	[alt_names]
	DNS.1 = localhost
	SSL

	openssl ecparam -name prime256v1 -genkey -out "${TMPDIR}/registry.key"
	openssl req -new -x509 -days 1 \
		-config "${TMPDIR}/registry.config" \
		-key    "${TMPDIR}/registry.key" \
		-out    "${TMPDIR}/registry.crt"

	# Start a local image registry.
	local image port registry
	registry=$(${container} run --detach --publish-all \
		--env='REGISTRY_HTTP_TLS_CERTIFICATE=/mnt/registry.crt' \
		--env='REGISTRY_HTTP_TLS_KEY=/mnt/registry.key' \
		--volume="${TMPDIR}:/mnt" \
		docker.io/library/registry:latest)
	# https://github.com/containers/podman/issues/8524
	push_trap_exit "echo -n 'Removing '; ${container} rm '${registry}'"
	push_trap_exit "echo -n 'Stopping '; ${container} stop '${registry}'"

	port=$(${container} inspect "${registry}" \
		--format='{{ (index .NetworkSettings.Ports "5000/tcp" 0).HostPort }}')
	image="localhost:${port}/postgres-operator-bundle:latest"

	cat > "${TMPDIR}/registries.conf" <<-TOML
	[[registry]]
	location = "localhost:${port}"
	insecure = true
	TOML

	# Build the bundle image and push it to the local registry.
	${container} run --rm \
		--device='/dev/fuse:rw' --network='host' --security-opt='seccomp=unconfined' \
		--volume="${TMPDIR}/registries.conf:/etc/containers/registries.conf.d/localhost.conf:ro" \
		--volume="${directory}:/mnt:delegated" \
		--workdir='/mnt' \
		quay.io/buildah/stable:latest \
			buildah build-using-dockerfile \
				--format='docker' --layers --tag="docker://${image}"

	local -a opm
	local opm_version
	opm_version=$(opm version)
	opm_version=$(sed -n 's#.*OpmVersion:"\([^"]*\)".*#\1# p' <<< "${opm_version}")
	# shellcheck disable=SC2206
	opm=(${container} run --rm
		--network='host'
		--volume="${TMPDIR}/registry.crt:/usr/local/share/ca-certificates/registry.crt:ro"
		--volume="${TMPDIR}:/mnt:delegated"
		--workdir='/mnt'
		quay.io/operator-framework/upstream-opm-builder:"${opm_version}"
			sh -ceu 'update-ca-certificates && exec "$@"' - opm)

	# Validate the bundle image in the local registry.
	# https://olm.operatorframework.io/docs/tasks/creating-operator-bundle/#validating-your-bundle
	"${opm[@]}" alpha bundle validate --image-builder='none' \
		--optional-validators='operatorhub,bundle-objects' \
		--tag="${image}"

	# Create an index database from the bundle image.
	"${opm[@]}" index add --bundles="${image}" --generate

	# drwxr-xr-x. 2 user user     22 database
	# -rw-r--r--. 1 user user 286720 database/index.db
	# -rw-r--r--. 1 user user    267 index.Dockerfile
}

validate_bundle_image "$@"
