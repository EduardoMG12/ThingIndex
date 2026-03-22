.PHONY: test test-unit test-integration test-e2e test-all docs-api \
	test-backend test-backend-unit test-backend-integration test-backend-e2e test-backend-smoke \
	test-frontend test-frontend-unit test-frontend-storybook test-frontend-e2e test-frontend-smoke test-frontend-e2e-install

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

test-backend:
	./scripts/test.sh backend-unit

test-backend-unit:
	./scripts/test.sh backend-unit

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
