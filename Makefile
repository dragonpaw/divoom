# Build + push + deploy the divoom dashboard to the Portainer instance on the
# home NAS. See docs/deploy.md for one-time setup (GHCR PAT, Portainer API
# key, stack ID).

IMAGE       := ghcr.io/dragonpaw/divoom
GHCR_USER   ?= dragonpaw
VERSION     := $(shell git describe --tags --always --dirty)
COMPOSE     := docker-compose.yml
ENV_FILE    := .env

PORTAINER_URL        ?= http://10.0.2.201:9000
PORTAINER_ENDPOINT   ?= 1
PORTAINER_API_KEY    ?= $(or $(PORTAINER_TOKEN),$(shell cat $(HOME)/.config/divoom/portainer-key 2>/dev/null))
PORTAINER_STACK_ID   ?= $(shell cat $(HOME)/.config/divoom/portainer-stack-id 2>/dev/null)

.PHONY: all build login push deploy

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

push: login
	podman push $(IMAGE):$(VERSION)
	podman push $(IMAGE):latest

# Editor-style stack update. Portainer CE has no webhook / git redeploy worth
# trusting, so we PUT the compose contents straight at:
#
#   PUT $PORTAINER_URL/api/stacks/$STACK_ID?endpointId=$ENDPOINT
#   X-API-Key: $PORTAINER_API_KEY
#   {
#     "stackFileContent": "<contents of docker-compose.yml>",
#     "env":              [{"name": "FOO", "value": "bar"}, ...],
#     "prune":            true,
#     "pullImage":        true
#   }
#
# `env` is sourced from .env (KEY=VALUE lines, blanks and # comments
# skipped). jq builds both the env array and the JSON envelope so that
# shell-special characters in values survive unescaped.
deploy: push
	@test -n "$(PORTAINER_API_KEY)"  || { echo "PORTAINER_API_KEY not set (env or ~/.config/divoom/portainer-key)"; exit 1; }
	@test -n "$(PORTAINER_STACK_ID)" || { echo "PORTAINER_STACK_ID not set (env or ~/.config/divoom/portainer-stack-id)"; exit 1; }
	@test -f $(COMPOSE)              || { echo "$(COMPOSE) missing"; exit 1; }
	@test -f $(ENV_FILE)             || { echo "$(ENV_FILE) missing (copy from .env.example)"; exit 1; }
	@body=$$(jq -n \
	    --rawfile compose $(COMPOSE) \
	    --rawfile envfile $(ENV_FILE) \
	    '{ stackFileContent: $$compose,
	       env: ($$envfile | split("\n")
	                       | map(select(test("^\\s*[^#\\s]") and contains("=")))
	                       | map(capture("^(?<name>[^=]+)=(?<value>.*)$$"))),
	       prune: true,
	       pullImage: true }'); \
	status=$$(curl -sS -o /tmp/portainer-deploy.out -w '%{http_code}' \
	    -X PUT "$(PORTAINER_URL)/api/stacks/$(PORTAINER_STACK_ID)?endpointId=$(PORTAINER_ENDPOINT)" \
	    -H "X-API-Key: $(PORTAINER_API_KEY)" \
	    -H "Content-Type: application/json" \
	    --data-binary "$$body"); \
	echo "portainer status: $$status"; \
	cat /tmp/portainer-deploy.out; echo; \
	case "$$status" in 2*) exit 0 ;; *) exit 1 ;; esac
