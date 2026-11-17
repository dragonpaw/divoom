# Build + push + deploy the divoom dashboard to plugger (10.0.2.203), where the
# Times Frame is USB-plugged. See docs/deploy.md. Portainer was retired
# 2026-11-16: the stack is plain Docker Compose in /srv/divoom on plugger
# (docker-compose.yml + stack.env), deployed over ssh.

IMAGE       := ghcr.io/dragonpaw/divoom
GHCR_USER   ?= dragonpaw
VERSION     := $(shell git describe --tags --always --dirty)
COMPOSE     := docker-compose.yml
ENV_FILE    := .env

DEPLOY_HOST          ?= ash@10.0.2.203
DEPLOY_DIR           := /srv/divoom
STACK_NAME           := divoom

.PHONY: all build login push deploy test vet lint fmt run probe render-out push-frame

# Default — ship both halves: deploy the plugger container stack (which
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

# Copy the compose file and .env (as stack.env, root 0600) to plugger, then pull
# + recreate there. .env on the dev box stays the source of truth for the
# container's env; an empty GITHUB_TOKEN is filled from GHCR_PAT/GITHUB_TOKEN.
deploy: push
	@test -f $(COMPOSE)  || { echo "$(COMPOSE) missing"; exit 1; }
	@test -f $(ENV_FILE) || { echo "$(ENV_FILE) missing (copy from .env.example)"; exit 1; }
	@tok="$$GHCR_PAT"; [ -z "$$tok" ] && tok="$$GITHUB_TOKEN"; \
	grep -v -E '^[[:space:]]*(#|$$)' $(ENV_FILE) | grep -v '^PORTAINER_' \
	    | sed "s|^GITHUB_TOKEN=$$|GITHUB_TOKEN=$$tok|" \
	    | ssh $(DEPLOY_HOST) 'sudo install -d $(DEPLOY_DIR) && sudo install -m 600 /dev/null $(DEPLOY_DIR)/stack.env && sudo tee $(DEPLOY_DIR)/stack.env >/dev/null'
	ssh $(DEPLOY_HOST) 'sudo tee $(DEPLOY_DIR)/docker-compose.yml >/dev/null' < $(COMPOSE)
	ssh $(DEPLOY_HOST) 'cd $(DEPLOY_DIR) && sudo docker compose -p $(STACK_NAME) --env-file stack.env pull && sudo docker compose -p $(STACK_NAME) --env-file stack.env up -d'

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
# env vars `make deploy` puts in plugger's stack.env — notably
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

# Push scene backgrounds + custom fonts to the frame via adb. Runs against
# whatever device adb sees — the dev box (USB-attached frame) or, since the
# serve container now ships with adb + /dev/bus/usb passthrough, the
# container on the NAS if the frame is USB-plugged into it. After any
# scene change, render-code change, or factory reset, run this from
# wherever the frame is currently cabled. The `push` target name is taken
# by the GHCR image push; this is the on-device push.
push-frame:
	$(withenv) go run ./cmd/divoom push
