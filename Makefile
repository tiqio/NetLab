.PHONY: bootstrap fmt lint test test-contract test-web test-e2e test-e2e-local test-e2e-target test-acceptance-schema test-frontend-acceptance test-integration test-recovery test-leaks test-security test-frontend-artifacts test-compliance acceptance acceptance-qemu acceptance-recovery acceptance-100 capture-candidate constitution-acceptance test-all build clean install

bootstrap:
	go mod download
	cd web && npm install

fmt:
	gofmt -w cmd internal
	cd web && npx prettier --write .

lint:
	go vet ./...
	cd web && npm run lint && npm run format:check

test:
	go test ./...

test-contract:
	go test ./tests/contract/...

test-web:
	cd web && npm test

test-e2e:
	$(MAKE) build
	cd web && NODE_PATH=$(CURDIR)/web/node_modules npm run test:e2e

test-e2e-local:
	$(MAKE) build
	cd web && NODE_PATH=$(CURDIR)/web/node_modules npm run test:e2e:local

test-e2e-target:
	cd web && NODE_PATH=$(CURDIR)/web/node_modules npm run test:e2e:target

test-acceptance-schema:
	go test ./tests/contract/... -run FrontendAcceptance -count=1

test-frontend-acceptance:
	./acceptance/frontend-acceptance.sh

test-integration:
	NETLAB_OWNERSHIP_DOMAIN=$${NETLAB_OWNERSHIP_DOMAIN:-local-integration} NETLAB_PRIVILEGED=$${NETLAB_PRIVILEGED:-0} go test ./tests/integration/... -count=1

test-recovery:
	NETLAB_PRIVILEGED=$${NETLAB_PRIVILEGED:-0} go test ./tests/recovery/... -count=1

test-leaks:
	NETLAB_OWNERSHIP_DOMAIN=$${NETLAB_OWNERSHIP_DOMAIN:-local-leak} NETLAB_PRIVILEGED=$${NETLAB_PRIVILEGED:-0} CYCLES=$${CYCLES:-100} go test ./tests/integration/... -run Leak -count=1

test-security:
	go test ./tests/security/... -count=1

test-frontend-artifacts:
	./scripts/check-frontend-artifacts.sh

test-compliance:
	./scripts/validate-compliance.sh

acceptance: constitution-acceptance

acceptance-qemu:
	CANDIDATE_ID=$${CANDIDATE_ID:?CANDIDATE_ID required} ./acceptance/qemu-acceptance.sh

acceptance-recovery:
	CANDIDATE_ID=$${CANDIDATE_ID:?CANDIDATE_ID required} ./acceptance/t225-service-restart.sh

acceptance-100:
	CANDIDATE_ID=$${CANDIDATE_ID:?CANDIDATE_ID required} NETLAB_PRIVILEGED=1 CYCLES=100 go test ./tests/integration/... -run Leak -count=1

capture-candidate:
	go run ./cmd/netlab-compliance capture-candidate --version "$${VERSION:-dev}" --candidate-id "$${CANDIDATE_ID:?CANDIDATE_ID required}" --binary "$${BINARY:-bin/netlabd}" --contracts specs/002-constitution-gap-closure/contracts --output "$${OUTPUT:--}"

constitution-acceptance:
	./scripts/run-constitution-acceptance.sh

test-all: lint test test-contract test-web test-security test-frontend-artifacts test-compliance test-integration test-recovery test-leaks build test-e2e

build:
	cd web && npm run build
	rm -rf internal/api/http/webdist && mkdir -p internal/api/http/webdist
	cp -R web/dist/. internal/api/http/webdist/
	go build -trimpath -ldflags "-X main.version=$${VERSION:-dev} -X main.candidateID=$${CANDIDATE_ID:-dev} -X main.binaryDigest=$${BINARY_DIGEST:-} -X main.contractDigest=$${CONTRACT_DIGEST:-} -X main.builtAt=$${BUILT_AT:-}" -o bin/netlabd ./cmd/netlabd

clean:
	rm -rf bin web/dist web/coverage web/playwright-report

install: build
	install -Dm0755 bin/netlabd /usr/local/bin/netlabd
	install -Dm0644 deploy/systemd/netlab.service /etc/systemd/system/netlab.service
	install -Dm0755 deploy/scripts/check-authority.sh /usr/local/libexec/netlab/check-authority.sh
	install -d -m0755 /usr/local/share/netlab/templates/qemu /usr/local/share/netlab/templates/docker
	install -m0644 templates/qemu/manifest.yaml /usr/local/share/netlab/templates/qemu/manifest.yaml
	install -m0644 templates/docker/manifest.yaml /usr/local/share/netlab/templates/docker/manifest.yaml
