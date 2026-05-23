# Build + push + deploy the divoom dashboard to the Portainer instance on the
# home NAS. See docs/deploy.md for one-time setup (GHCR PAT, Portainer API key).

IMAGE       := ghcr.io/dragonpaw/divoom
GHCR_USER   ?= dragonpaw
VERSION     := $(shell git describe --tags --always --dirty)
COMPOSE     := docker-compose.yml
ENV_FILE    := .env

PORTAINER_URL        ?= http://10.0.2.201:19900
PORTAINER_ENDPOINT   ?= 1
PORTAINER_API_KEY    ?= $(or $(PORTAINER_TOKEN),$(shell cat $(HOME)/.config/divoom/portainer-key 2>/dev/null))
STACK_NAME           ?= divoom

.PHONY: all build login push deploy stacks test vet lint fmt run probe render-out push-frame

# Default — ship both halves: deploy the NAS container stack (which
# transitively builds + pushes the image) and adb-push fresh scene
# bgs + fonts to the frame. push-frame depends on a USB-attached
# device; run `make deploy` alone from a host without that.
all: deploy push-frame

build:
	podman build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

# Auto-login if GHCR_PAT or GITHUB_TOKEN is in the env; otherwise assume
# `podman login ghcr.io` was done previously (or that podman has a cached
# credential). The same PAT serves both GHCR push and the github-activity
# scene, so users may keep it in either env var.
login:
	@token="$$GHCR_PAT"; [ -z "$$token" ] && token="$$GITHUB_TOKEN"; \
	if [ -n "$$token" ]; then \
	    echo "$$token" | podman login ghcr.io -u $(GHCR_USER) --password-stdin; \
	else \
	    echo "neither GHCR_PAT nor GITHUB_TOKEN set — relying on existing podman login session"; \
	fi

push: build login
	podman push $(IMAGE):$(VERSION)
	podman push $(IMAGE):latest

# List Portainer stacks (id, name, endpoint).
stacks:
	@test -n "$(PORTAINER_API_KEY)" || { echo "PORTAINER_API_KEY not set (env or ~/.config/divoom/portainer-key)"; exit 1; }
	@curl -sS -H "X-API-Key: $(PORTAINER_API_KEY)" \
	    "$(PORTAINER_URL)/api/stacks" \
	    | jq -r '["ID","NAME","ENDPOINT"], (.[] | [.Id, .Name, .EndpointId]) | @tsv' \
	    | column -t -s "$$(printf '\t')"

# Find the stack named $(STACK_NAME); create it if missing, update it if
# present. Portainer CE has no webhook / git redeploy worth trusting, so
# we send compose contents inline either way:
#
#   POST $URL/api/stacks/create/standalone/string?endpointId=$ENDPOINT  (create)
#   PUT  $URL/api/stacks/$ID?endpointId=$ENDPOINT                       (update)
#
# `env` is sourced from .env (KEY=VALUE lines; blanks and # comments skipped).
# jq builds both the env array and the JSON envelope so that shell-special
# characters in values survive unescaped.
deploy: push
	@test -n "$(PORTAINER_API_KEY)" || { echo "PORTAINER_API_KEY not set (env or ~/.config/divoom/portainer-key)"; exit 1; }
	@test -f $(COMPOSE)             || { echo "$(COMPOSE) missing"; exit 1; }
	@test -f $(ENV_FILE)            || { echo "$(ENV_FILE) missing (copy from .env.example)"; exit 1; }
	@stack_id=$$(curl -sS -H "X-API-Key: $(PORTAINER_API_KEY)" "$(PORTAINER_URL)/api/stacks" \
	    | jq -r --arg n "$(STACK_NAME)" '.[] | select(.Name == $$n) | .Id' | head -1); \
	tok="$$GHCR_PAT"; [ -z "$$tok" ] && tok="$$GITHUB_TOKEN"; \
	env_json=$$(jq -n --rawfile envfile $(ENV_FILE) --arg tok "$$tok" \
	    '$$envfile | split("\n") | map(select(test("^\\s*[^#\\s]") and contains("="))) \
	                            | map(capture("^(?<name>[^=]+)=(?<value>.*)$$")) \
	                            | map(if .name == "GITHUB_TOKEN" and (.value | length) == 0 and ($$tok | length) > 0 \
	                                  then .value = $$tok else . end)'); \
	if [ -z "$$stack_id" ]; then \
	    echo "creating new stack '$(STACK_NAME)'"; \
	    body=$$(jq -n --rawfile compose $(COMPOSE) --argjson env "$$env_json" --arg name "$(STACK_NAME)" \
	        '{ name: $$name, stackFileContent: $$compose, env: $$env }'); \
	    status=$$(curl -sS -o /tmp/portainer-deploy.out -w '%{http_code}' \
	        -X POST "$(PORTAINER_URL)/api/stacks/create/standalone/string?endpointId=$(PORTAINER_ENDPOINT)" \
	        -H "X-API-Key: $(PORTAINER_API_KEY)" -H "Content-Type: application/json" \
	        --data-binary "$$body"); \
	else \
	    echo "updating existing stack '$(STACK_NAME)' (id=$$stack_id)"; \
	    body=$$(jq -n --rawfile compose $(COMPOSE) --argjson env "$$env_json" \
	        '{ stackFileContent: $$compose, env: $$env, prune: true, pullImage: true }'); \
	    status=$$(curl -sS -o /tmp/portainer-deploy.out -w '%{http_code}' \
	        -X PUT "$(PORTAINER_URL)/api/stacks/$$stack_id?endpointId=$(PORTAINER_ENDPOINT)" \
	        -H "X-API-Key: $(PORTAINER_API_KEY)" -H "Content-Type: application/json" \
	        --data-binary "$$body"); \
	fi; \
	echo "portainer status: $$status"; \
	cat /tmp/portainer-deploy.out; echo; \
	case "$$status" in 2*) exit 0 ;; *) exit 1 ;; esac

test:
	go test ./...

vet:
	go vet ./...

# Optional — only if golangci-lint is on PATH.
lint:
	@command -v golangci-lint >/dev/null && golangci-lint run \
	    || echo "golangci-lint not installed; skipping"

fmt:
	gofmt -w .

# `withenv` sources .env into the recipe shell so locally-running
# subcommands (serve / probe / render / push-frame) see the same
# env vars `make deploy` injects into the Portainer stack — notably
# NASA_API_KEY, GITHUB_TOKEN, WORDNIK_API_KEY. Recipes invoke it
# with $(withenv) at the front of the command line.
withenv = set -a; [ -f $(ENV_FILE) ] && . ./$(ENV_FILE); set +a;

# Run the daemon locally against the configured frame.
run:
	$(withenv) go run ./cmd/divoom serve

probe:
	$(withenv) go run ./cmd/divoom probe

# Render every scene background JPG to ./dist/scenes/ for inspection.
render-out:
	$(withenv) go run ./cmd/divoom render

# Push scene backgrounds + custom fonts to the frame via adb (USB host
# only — the NAS serve container can't do this). After any scene
# change, render-code change, or factory reset, run this from the
# dev box. The `push` target name is taken by the GHCR image push;
# this is the on-device push.
push-frame:
	$(withenv) go run ./cmd/divoom push
