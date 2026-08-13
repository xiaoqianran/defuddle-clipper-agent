.PHONY: build build-agent build-extension test test-agent typecheck clean

build: build-agent build-extension

build-agent:
	cd apps/agent && go build ./cmd/clipper-agent

build-extension:
	npm run build

test: test-agent typecheck

test-agent:
	cd apps/agent && go test ./...

typecheck:
	npm run typecheck

clean:
	cd apps/agent && go clean
	npm run clean
