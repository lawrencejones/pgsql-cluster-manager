PROG=bin/pgcm bin/pgcm-acceptance
VERSION := $(shell git rev-parse --short HEAD)-dev
BUILD_COMMAND := go build -ldflags "-X github.com/gocardless/pgsql-cluster-manager/pkg/cmd.Version=$(VERSION)"

# .PHONY: build build-integration generate test clean postgres-member-docker publish-dockerfile publish-circleci-dockerfile
.PHONY: all generate clean

all: $(PROG)

generate:
	go generate ./...

# Specific linux build target, making it easy to work with the docker acceptance
# tests on OSX
bin/%.linux_amd64:
	GOOS=linux GOARCH=amd64 $(BUILD_COMMAND) -o $@ cmd/$*/$*.go

bin/%:
	$(BUILD_COMMAND) -o $@ cmd/$*/$*.go

test:
	go test ./... -v

test-integration: docker-postgres-member bin/pgcm-acceptance bin/pgcm.linux_amd64
	bin/pgcm-acceptance --workspace $$(pwd)

clean:
	rm -rvf dist $(PROG) $(PROG:%=%.linux_amd64)

docker-base: Dockerfile
	docker build -t gocardless/pgsql-cluster-manager-base:v1 .

docker-postgres-member: docker/postgres-member/Dockerfile
	docker build -t gocardless/postgres-member:v1 docker/postgres-member

docker-circleci: .circleci/Dockerfile
	docker build -t gocardless/pgsql-cluster-manager-circleci:v1 .circleci

publish-base: docker-base
	docker push gocardless/pgsql-cluster-manager-base:v1

publish-postgres-member: docker-postgres-member
	docker push gocardless/postgres-member:v1

publish-circleci: docker-circleci
	docker push gocardless/pgsql-cluster-manager-circleci:v1
