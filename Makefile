APP_NAME := go-sit

.PHONY: mocks
mocks:
	go generate ./...

.PHONY: tests
tests:
	go test -v -count=1 -cover -race ./...

.PHONY: docker-go-lint
docker-go-lint:
	docker run \
		--rm \
		--name $(APP_NAME)-docker-go-lint \
		--volume $(PWD):/src/github.com/lucidhq/$(APP_NAME) \
		--workdir /src/github.com/lucidhq/$(APP_NAME) \
		golangci/golangci-lint:v1.30.0 \
		/bin/bash -c "git config --global url.'https://$(GITHUB_TOKEN):@github.com/'.insteadOf 'https://github.com/' && \
			golangci-lint run -v"

.PHONY: docker-go-tests
docker-go-tests:
	docker run \
		--rm \
		--name $(APP_NAME)-docker-go-tests \
		--volume $(PWD):/src/github.com/lucidhq/$(APP_NAME) \
		--workdir /src/github.com/lucidhq/$(APP_NAME) \
		--env GOSUMDB=off \
		golang:1.15-buster \
		/bin/sh -c "apt-get update && apt-get --assume-yes upgrade ca-certificates && \
			git config --global url.'https://$(GITHUB_TOKEN):@github.com/'.insteadOf 'https://github.com/' && \
			go mod download && \
			make tests"
