.PHONY: build
build: bin/benchmarker

.PHONY: test
test:
	go test ./...

.PHONY: dev
dev: build
	./bin/benchmarker

.PHONY: run
run: build
	./bin/benchmarker --stage=prod --request-timeout=10s

bin/benchmarker: $(shell find . -name '*.go' -print)
	go build -buildvcs=false -o $@ .

.PHONY: run-ci
run-ci: build
	./bin/benchmarker --target-host=localhost --request-timeout=30s --initialize-request-timeout=60s --exit-error-on-fail=true --stage=prod --max-parallelism=10

.PHONY: clean
clean:
	[ -f ./bin/benchmarker ] && rm ./bin/benchmarker
