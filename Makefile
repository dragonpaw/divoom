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

.PHONY: all build login push deploy stacks

all: build push deploy

build:
	podman build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

# Auto-login if GHCR_PAT is in the env; otherwise assume `podman login ghcr.io`
# was done previously (or that podman has a cached credential).
login:
	@if [ -n "$(GHCR_PAT)" ]; then \
	    echo "$(GHCR_PAT)" | podman login ghcr.io -u $(GHCR_USER) --password-stdin; \
	else \
	    echo "GHCR_PAT not set — relying on existing podman login session"; \
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
	env_json=$$(jq -n --rawfile envfile $(ENV_FILE) \
	    '$$envfile | split("\n") | map(select(test("^\\s*[^#\\s]") and contains("=")))
	                            | map(capture("^(?<name>[^=]+)=(?<value>.*)$$"))'); \
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
