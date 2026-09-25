include .env
export

PORTS = 8001 8002 8003

LSOF_ARGS = $(foreach port,$(PORTS),-i:$(port))

.PHONY: start-backends stop-backends clean test

test:
	go test -race -v ./...

start-backends: clean
	@for port in $(PORTS); do \
		echo "Starting backend on port $$port..."; \
		go run ./cmd/mockbackend/main.go http://localhost:$$port >> backend_$$port.log 2>&1 & \
	done
	@echo "All mock backends started!"

stop-backends:
	@echo "Clearing ports $(PORTS)..."
	@# The minus sign (-) at the start tells Make to ignore errors if lsof finds nothing
	@-kill -9 $$(lsof -t $(LSOF_ARGS) -sTCP:LISTEN) 2>/dev/null || true
	@echo "All backends stopped."

clean: stop-backends
	rm -f backend_*.log
	@echo "Cleaned up log files."