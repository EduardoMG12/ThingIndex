.PHONY: bootstrap doctor stack-up stack-dev stack-core stack-down stack-dev-down stack-ps stack-dev-ps stack-logs stack-dev-logs test test-unit test-integration test-e2e test-all docs-api benchmark-indexing \
	test-backend test-backend-unit test-backend-benchmark test-backend-integration test-backend-e2e test-backend-smoke \
	test-frontend test-frontend-unit test-frontend-storybook test-frontend-e2e test-frontend-smoke test-frontend-e2e-install

bootstrap:
	./scripts/bootstrap.sh all

doctor:
	./scripts/doctor.sh

stack-up:
	./scripts/stack.sh up

stack-dev:
	./scripts/stack.sh dev

stack-core:
	./scripts/stack.sh core

stack-down:
	./scripts/stack.sh down

stack-dev-down:
	./scripts/stack.sh dev-down

stack-ps:
	./scripts/stack.sh ps

stack-dev-ps:
	./scripts/stack.sh dev-ps

stack-logs:
	./scripts/stack.sh logs

stack-dev-logs:
	./scripts/stack.sh dev-logs

test: test-unit

test-unit:
	./scripts/test.sh unit

test-integration:
	./scripts/test.sh backend-integration

test-e2e:
	./scripts/test.sh e2e

test-all:
	./scripts/test.sh all

docs-api:
	./scripts/generate-api-docs.sh

benchmark-indexing:
	./scripts/indexing-benchmark.sh

test-backend:
	./scripts/test.sh backend-unit

test-backend-unit:
	./scripts/test.sh backend-unit

test-backend-benchmark:
	./scripts/test.sh backend-benchmark

test-backend-integration:
	./scripts/test.sh backend-integration

test-backend-e2e:
	./scripts/test.sh backend-e2e

test-backend-smoke:
	./scripts/test.sh backend-smoke

test-frontend:
	./scripts/test.sh frontend-unit

test-frontend-unit:
	./scripts/test.sh frontend-unit

test-frontend-storybook:
	./scripts/test.sh frontend-storybook

test-frontend-e2e:
	./scripts/test.sh frontend-e2e

test-frontend-smoke:
	./scripts/test.sh frontend-smoke

test-frontend-e2e-install:
	./scripts/test.sh frontend-e2e-install
