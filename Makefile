VERSION = "unset"
DATE=$(shell date -u +%Y-%m-%d-%H:%M:%S-%Z)

ifndef GOLANG_VERSION
override GOLANG_VERSION = 1.14
endif

.PHONY: gobuildcache

.PHONY: build
build: ## build the athens proxy
	go build -o ./cmd/proxy/proxy ./cmd/proxy

.PHONY: build-ver
build-ver: ## build the athens proxy with version number
	GO111MODULE=on CGO_ENABLED=0 GOPROXY="https://proxy.golang.org" go build -ldflags "-X github.com/gomods/athens/pkg/build.version=$(VERSION) -X github.com/gomods/athens/pkg/build.buildDate=$(DATE)" -o athens ./cmd/proxy

athens:
	$(MAKE) build-ver

# The build-image step creates a docker image that has all the tools required
# to perform some CI build steps, instead of relying on them being installed locally
.PHONY: build-image
build-image:
	docker build -t athens-build ./scripts/build-image

.PHONY: run
run: ## run the athens proxy with dev configs
	cd ./cmd/proxy && go run . -config_file ../../config.dev.toml

.PHONY: run-docker
run-docker:
	docker-compose -p athensdockerdev build --build-arg GOLANG_VERSION=${GOLANG_VERSION} dev
	docker-compose -p athensdockerdev up -d dev

.PHONY: run-docker-teardown
run-docker-teardown:
	docker-compose -p athensdockerdev down

.PHONY: docs
docs: ## build the docs docker image
	docker build -t gomods/hugo -f docs/Dockerfile .

.PHONY: setup-dev-env
setup-dev-env:
	./scripts/get_dev_tools.sh
	$(MAKE) dev

.PHONY: verify
verify: ## verify athens codebase
	./scripts/check_gofmt.sh
	./scripts/check_golint.sh
	./scripts/check_deps.sh
	./scripts/check_conflicts.sh

.PHONY: test-unit
test-unit: ## run unit tests with race detector and code coverage enabled
	./scripts/test_unit.sh

.PHONY: test-unit-docker
test-unit-docker: ## run unit tests with docker
	docker-compose -p athensunit build --build-arg GOLANG_VERSION=${GOLANG_VERSION} testunit	
	docker-compose -p athensunit up --exit-code-from=testunit testunit
	docker-compose -p athensunit down

.PHONY: test-all
test-all:
	./scripts/test_all.sh

test-all-docker: testdeps
	docker-compose -p athenstest up --build --exit-code-from=testall testall
	$(MAKE) test-teardown

.PHONY: test-e2e
test-e2e:
	cd e2etests && go test --tags e2etests

.PHONY: test-e2e-docker
test-e2e-docker:
	docker-compose -p athense2e build --build-arg GOLANG_VERSION=${GOLANG_VERSION} teste2e
	docker-compose -p athense2e up --exit-code-from=teste2e teste2e
	docker-compose -p athense2e down

.PHONY: docker
docker: proxy-docker

.PHONY: proxy-docker
proxy-docker:
	docker build -t gomods/athens -f cmd/proxy/Dockerfile --build-arg GOLANG_VERSION=${GOLANG_VERSION} .

.PHONY: docker-push
docker-push:
	./scripts/push-docker-images.sh

.PHONY: charts-push
charts-push: build-image
	docker run --rm -it \
	-v `pwd`:/go/src/github.com/gomods/athens \
	-e AZURE_STORAGE_CONNECTION_STRING \
	-e CHARTS_REPO \
	athens-build ./scripts/push-helm-charts.sh

bench:
	./scripts/benchmark.sh

.PHONY: alldeps
alldeps:
	docker-compose -p athensdev up -d mongo minio jaeger
	echo "sleeping for a bit to wait for the DB to come up"
	sleep 5

.PHONY: no-static-ports
no-static-ports: export MONGO_27017 = 0
no-static-ports: export MINIO_9000 = 0
no-static-ports: export JAEGER_14268 = 0
no-static-ports: export JAEGER_9411 = 0
no-static-ports: export JAEGER_5775 = 0
no-static-ports: export JAEGER_6831 = 0
no-static-ports: export JAEGER_6832 = 0
no-static-ports: export JAEGER_5778 = 0
no-static-ports: export JAEGER_16686 = 0
no-static-ports: export REDIS_6379 = 0
no-static-ports: export REDIS_SENTINEL_26379 = 0
no-static-ports: export PROTECTEDREDIS_6380 = 0
no-static-ports: export ETCD0_2379 = 0
no-static-ports: export ETCD1_2379 = 0
no-static-ports: export ETCD2_2379 = 0
no-static-ports: export AZURITE_10000 = 0

testdeps: no-static-ports bin/composeconfig
	docker-compose -p athenstest up -d \
mongo minio jaeger mysql postgres etcd0 etcd1 etcd2 redis redis-sentinel protectedredis azurite
	bin/composeconfig -p athenstest -config-file config.test.toml

.PHONY: dev
dev:
	docker-compose -p athensdev up -d mongo
	docker-compose -p athensdev up -d minio

.PHONY: down
down:
	docker-compose -p athensdev down -v

.PHONY: dev-teardown
dev-teardown:
	docker-compose -p athensdev down -v

.PHONY: test-teardown
test-teardown:
	docker-compose -p athenstest down -v
	rm config.test.toml

.PHONY: clean
clean: ## delete all locally-built artefacts (not including docker images)
	rm -f athens cmd/proxy/proxy bin/composeconfig

.PHONY: help
help: ## display help page
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: deploy-gae
deploy-gae:
	cd scripts/gae && gcloud app deploy

bin/composeconfig: gobuildcache
	go build -o $@ ./cmd/composeconfig
