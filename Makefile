# Build + deploy the divoom dashboard to plugger (10.0.2.203), where the Times
# Frame is USB-plugged. It runs NATIVELY there since 2026-11-16 (systemd
# divoom.service, deploy/), not in a container. See docs/deploy.md.

ENV_FILE    := .env
BIN         := dist/divoom

DEPLOY_HOST          ?= ash@10.0.2.203
DEPLOY_DIR           := /srv/divoom
STACK_NAME           := divoom

.PHONY: all build deploy test vet lint fmt run probe render-out push-frame

# Default — ship both halves: deploy the daemon to plugger and adb-push fresh
# scene bgs + fonts to the frame (it's USB-attached to plugger).
all: deploy push-frame

# Static binary, same flags the old Dockerfile used.
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/divoom

# Copy the binary, fonts/, device-files/, the unit and the udev rule to plugger,
# install them and restart. .env, when present on this box, becomes plugger's
# /srv/divoom/stack.env (root 0600); without it the host's stack.env is kept.
# fonts/ must be populated first (scripts/download-fonts.sh).
deploy: build
	@ls fonts/*.ttf >/dev/null 2>&1 || { echo "fonts/ is empty: run scripts/download-fonts.sh"; exit 1; }
	@if [ -f $(ENV_FILE) ]; then \
	    tok="$$GHCR_PAT"; [ -z "$$tok" ] && tok="$$GITHUB_TOKEN"; \
	    grep -v -E '^[[:space:]]*(#|$$)' $(ENV_FILE) | grep -v '^PORTAINER_' \
	        | sed "s|^GITHUB_TOKEN=$$|GITHUB_TOKEN=$$tok|" \
	        | ssh $(DEPLOY_HOST) 'sudo install -d $(DEPLOY_DIR) && sudo install -m 600 /dev/null $(DEPLOY_DIR)/stack.env && sudo tee $(DEPLOY_DIR)/stack.env >/dev/null'; \
	else echo "no $(ENV_FILE) here: keeping $(DEPLOY_HOST):$(DEPLOY_DIR)/stack.env"; fi
	rsync -az --delete --rsync-path="sudo rsync" $(BIN) fonts device-files deploy/ $(DEPLOY_HOST):/opt/divoom/
	ssh $(DEPLOY_HOST) 'getent passwd divoom >/dev/null || sudo useradd -r -d /var/lib/divoom -s /usr/bin/nologin divoom; \
	    sudo chown -R root:root /opt/divoom && sudo chmod -R a+rX /opt/divoom && \
	    sudo install -m 644 /opt/divoom/divoom.service /etc/systemd/system/divoom.service && \
	    sudo install -m 644 /opt/divoom/70-divoom-frame.rules /etc/udev/rules.d/70-divoom-frame.rules && \
	    sudo udevadm control --reload && sudo udevadm trigger --subsystem-match=usb --attr-match=idVendor=18d1 && \
	    sudo systemctl daemon-reload && sudo systemctl enable divoom.service >/dev/null 2>&1; \
	    sudo systemctl restart divoom.service'

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

# Push scene bgs + fonts to the frame. Runs ON plugger (the frame is USB-attached
# there) as the service's user with the service's env, after `make deploy`.
push-frame:
	ssh $(DEPLOY_HOST) 'sudo systemd-run --wait --pipe --collect --quiet -p User=divoom \
	    -p WorkingDirectory=/opt/divoom -p EnvironmentFile=$(DEPLOY_DIR)/stack.env \
	    -p Environment="XDG_CACHE_HOME=/var/cache HOME=/var/lib/divoom" /opt/divoom/divoom push'
