.PHONY: help
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'match($$0, /^([a-zA-Z_\/-]+):.*?## (.*)$$/, m) {printf "  \033[36m%-30s\033[0m %s\n", m[1], m[2]}' $(MAKEFILE_LIST) | sort

BASE_CONTAINER_IMAGE_NAME?=registry.fedoraproject.org/fedora
BASE_CONTAINER_IMAGE_TAG?=40
BASE_CONTAINER_IMAGE?=${BASE_CONTAINER_IMAGE_NAME}:${BASE_CONTAINER_IMAGE_TAG}

CONTAINERFILE=Containerfile
CONTAINER_IMAGE?=osbuild-images_$(shell echo $(BASE_CONTAINER_IMAGE) | tr '/:.' '_')
CONTAINER_EXECUTABLE?=podman

container_built_$(CONTAINER_IMAGE).info: $(CONTAINERFILE) Schutzfile test/ go.mod go.sum # internal rule to build the container only if needed
	$(CONTAINER_EXECUTABLE) build --build-arg BASE_CONTAINER_IMAGE="${BASE_CONTAINER_IMAGE}" \
	                              --tag $(CONTAINER_IMAGE) \
	                              -f $(CONTAINERFILE) .
	echo "Container last built on" > $@
	date >> $@

.PHONY: gh-action-test
gh-action-test: container_built_$(CONTAINER_IMAGE).info ## run all tests in a container (see BASE_CONTAINER_IMAGE_* in Makefile)
	podman run -v .:/app:z --rm -t $(CONTAINER_IMAGE) make test

.PHONY: test
test: ## run all tests locally
	go test -race  ./...
	# Run depsolver tests with force-dnf to make sure it's not skipped for any reason
	go test -race ./pkg/dnfjson/... -force-dnf

clean: ## remove all build files
	rm -f container_built*.info
