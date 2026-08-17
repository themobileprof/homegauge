# HomeGauge local run. API must start from this directory so .env loads.
.DEFAULT_GOAL := help

ROOT     := $(CURDIR)
RUN      := $(ROOT)/.run
API_BIN  := $(RUN)/homegauge-api
SEED_BIN := $(RUN)/homegauge-seed
API_LOG  := $(RUN)/api.log
WEB_LOG  := $(RUN)/web.log
API_PID  := $(RUN)/api.pid
WEB_PID  := $(RUN)/web.pid
API_URL  := http://127.0.0.1:8080
WEB_URL  := http://127.0.0.1:3000

.PHONY: help start stop restart status seed build logs start-api start-web stop-api stop-web fix-swc

help:
	@echo "HomeGauge"
	@echo "  make start     build API if needed, start API + web in the background"
	@echo "  make stop      stop API + web"
	@echo "  make restart   stop then start"
	@echo "  make status    what is listening on :8080 and :3000"
	@echo "  make logs      follow API and web logs (Ctrl-C to detach; servers keep running)"
	@echo "  make seed      reset demo users/products"
	@echo "  make build     compile API and seed binaries into .run/"
	@echo "  make fix-swc   restore the vendored Next SWC binary if next dev bus-errors"

$(RUN):
	mkdir -p $(RUN)

build: | $(RUN)
	go -C backend build -o $(API_BIN) ./cmd/api
	go -C backend build -o $(SEED_BIN) ./cmd/seed

start: start-api start-web
	@echo
	@echo "Web  $(WEB_URL)"
	@echo "API  $(API_URL)/health"

start-api: build
	@if ss -H -ltn sport = :8080 | grep -q .; then \
		echo "API already running on :8080"; \
	else \
		cd $(ROOT) && nohup $(API_BIN) >> $(API_LOG) 2>&1 & echo $$! > $(API_PID); \
		echo "API starting (pid $$(cat $(API_PID)))"; \
		for i in 1 2 3 4 5 6 7 8 9 10 11 12; do \
			curl -fsS $(API_URL)/health >/dev/null 2>&1 && break; \
			sleep 0.25; \
		done; \
		curl -fsS $(API_URL)/health >/dev/null 2>&1 && echo "API ready $(API_URL)/health" || echo "API did not become healthy — see $(API_LOG)"; \
	fi

start-web: | $(RUN)
	@if ss -H -ltn sport = :3000 | grep -q .; then \
		echo "Web already running on :3000"; \
	else \
		cd $(ROOT)/frontend && nohup npm run dev >> $(WEB_LOG) 2>&1 & echo $$! > $(WEB_PID); \
		echo "Web starting (pid $$(cat $(WEB_PID)))"; \
		for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24; do \
			ss -H -ltn sport = :3000 | grep -q . && break; \
			sleep 0.5; \
		done; \
		ss -H -ltn sport = :3000 | grep -q . && echo "Web ready $(WEB_URL)" || echo "Web did not bind :3000 yet — see $(WEB_LOG)"; \
	fi

stop: stop-api stop-web
	@echo "Stopped."

stop-api:
	@$(call stop_pidtree,$(API_PID))
	@fuser -k 8080/tcp >/dev/null 2>&1 || true
	@$(call wait_port_free,8080)
	@echo "API stopped"

stop-web:
	@$(call stop_pidtree,$(WEB_PID))
	@fuser -k 3000/tcp >/dev/null 2>&1 || true
	@$(call wait_port_free,3000)
	@echo "Web stopped"

restart: stop start

status:
	@echo -n "API :8080  "; ss -H -ltn sport = :8080 | grep -q . && echo "up  $(API_URL)/health" || echo "down"
	@echo -n "Web :3000  "; ss -H -ltn sport = :3000 | grep -q . && echo "up  $(WEB_URL)" || echo "down"

seed: build
	cd $(ROOT) && $(SEED_BIN)

logs: | $(RUN)
	@touch $(API_LOG) $(WEB_LOG)
	tail -n 80 -F $(API_LOG) $(WEB_LOG)

fix-swc:
	tar -xzf $(ROOT)/frontend/.vendor/swc-linux-x64-gnu-15.5.23.tgz -C /tmp
	rm -rf $(ROOT)/frontend/node_modules/@next/swc-linux-x64-gnu
	mv /tmp/package $(ROOT)/frontend/node_modules/@next/swc-linux-x64-gnu
	@echo "Restored SWC binary. Run make start (or make start-web) again."

define stop_pidtree
	kill_descendants() { \
		for c in $$(pgrep -P "$$1" 2>/dev/null); do kill_descendants "$$c"; done; \
		kill "$$1" 2>/dev/null || true; \
	}; \
	if [ -f $(1) ]; then \
		pid=$$(cat $(1)); \
		if kill -0 "$$pid" 2>/dev/null; then \
			kill_descendants "$$pid"; \
			sleep 0.2; \
			kill -9 "$$pid" 2>/dev/null || true; \
		fi; \
		rm -f $(1); \
	fi
endef

define wait_port_free
	for i in 1 2 3 4 5 6 7 8 9 10; do \
		ss -H -ltn sport = :$(1) | grep -q . || exit 0; \
		sleep 0.2; \
	done
endef
