.PHONY: all build dev test lint clean install-deps build-macos build-linux \
        build-deb build-rpm test-unit test-integration

# ── Variables ──────────────────────────────────────────────────────────────────
APP_NAME    := warpture
VERSION     := 1.0.0
BUILD_DIR   := dist
GO_VERSION  := 1.21
NODE_VERSION:= 18

# ── Default ────────────────────────────────────────────────────────────────────
all: build

# ── Install dependencies ───────────────────────────────────────────────────────
install-deps:
	@echo "→ Installing GUI dependencies..."
	cd services/warp-gui && npm ci
	@echo "→ Installing tunnel-agent dependencies..."
	cd services/tunnel-agent && go mod download
	@echo "→ Installing process-monitor dependencies..."
	cd services/process-monitor && pip install -r requirements.txt
	@echo "✓ All dependencies installed"

# ── Development mode ───────────────────────────────────────────────────────────
dev:
	@echo "→ Starting all services in dev mode..."
	docker-compose up --build

dev-gui:
	cd services/warp-gui && npm run dev

dev-agent:
	cd services/tunnel-agent && go run cmd/server/main.go

dev-monitor:
	cd services/process-monitor && python monitor.py

# ── Build ──────────────────────────────────────────────────────────────────────
build: build-agent build-monitor build-gui
	@echo "✓ All services built"

build-gui:
	@echo "→ Building warp-gui..."
	cd services/warp-gui && npm run build

build-agent:
	@echo "→ Building tunnel-agent..."
	cd services/tunnel-agent && go build -ldflags="-s -w -X main.Version=$(VERSION)" -o ../../$(BUILD_DIR)/tunnel-agent ./cmd/server/

build-monitor:
	@echo "→ Building process-monitor..."
	cd services/process-monitor && \
		pip install pyinstaller && \
		pyinstaller --onefile --name process-monitor monitor.py && \
		cp dist/process-monitor ../../$(BUILD_DIR)/

# ── Platform builds ────────────────────────────────────────────────────────────
build-macos: build
	@echo "→ Building macOS application bundle..."
	cd services/warp-gui && npm run build:mac
	./scripts/build-macos-bundle.sh $(VERSION)
	@echo "✓ macOS build complete: $(BUILD_DIR)/WARPture-$(VERSION).dmg"

build-linux: build
	@echo "→ Building Linux packages..."
	$(MAKE) build-deb build-rpm
	@echo "✓ Linux builds complete"

build-deb: build
	@echo "→ Building .deb package..."
	./scripts/build-deb.sh $(VERSION)
	@echo "✓ DEB: $(BUILD_DIR)/warpture_$(VERSION)_amd64.deb"

build-rpm: build
	@echo "→ Building .rpm package..."
	./scripts/build-rpm.sh $(VERSION)
	@echo "✓ RPM: $(BUILD_DIR)/warpture-$(VERSION).x86_64.rpm"

# ── Tests ──────────────────────────────────────────────────────────────────────
test: test-unit test-integration

test-unit:
	@echo "→ Running unit tests..."
	cd services/tunnel-agent && go test ./... -v -race
	cd services/warp-gui && npm test -- --watchAll=false
	cd services/process-monitor && python -m pytest tests/ -v
	@echo "✓ Unit tests complete"

test-integration:
	@echo "→ Running integration tests..."
	./scripts/integration-test.sh
	@echo "✓ Integration tests complete"

# ── Lint ───────────────────────────────────────────────────────────────────────
lint:
	@echo "→ Linting..."
	cd services/warp-gui && npm run lint
	cd services/tunnel-agent && golangci-lint run ./...
	cd services/process-monitor && flake8 . --max-line-length=100
	@echo "✓ Lint complete"

# ── Clean ──────────────────────────────────────────────────────────────────────
clean:
	rm -rf $(BUILD_DIR)/
	cd services/warp-gui && rm -rf node_modules/ dist/ .cache/
	cd services/tunnel-agent && rm -f tunnel-agent
	cd services/process-monitor && rm -rf __pycache__/ dist/ build/ *.spec
	@echo "✓ Clean complete"

# ── Docker ─────────────────────────────────────────────────────────────────────
docker-build:
	docker-compose build

docker-push:
	docker-compose push

# ── Help ───────────────────────────────────────────────────────────────────────
help:
	@echo "WARPture Build System"
	@echo "─────────────────────"
	@echo "  make install-deps   Install all dependencies"
	@echo "  make dev            Start all services (Docker)"
	@echo "  make build          Build all services"
	@echo "  make build-macos    Build macOS DMG"
	@echo "  make build-linux    Build Linux DEB + RPM"
	@echo "  make test           Run all tests"
	@echo "  make lint           Lint all code"
	@echo "  make clean          Remove build artifacts"
