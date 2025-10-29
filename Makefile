APP_NAME=haunteed
SRC=./cmd/haunteed
VERSION=$(shell grep '^version:' snap/snapcraft.yaml | cut -d' ' -f2)
LDFLAGS=-ldflags "-X 'main.version=$(VERSION)'"

all: build

build: linux windows darwin

linux: linux-amd64 linux-arm64

linux-amd64:
	@echo "Building Linux amd64..."
	GOOS=linux GOARCH=amd64   go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64   $(SRC)

linux-arm64:
	@echo "Building Linux arm64..."
	GOOS=linux GOARCH=arm64   go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-arm64   $(SRC)

windows: windows-amd64 windows-arm64

windows-amd64:
	@echo "Building Windows amd64..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe $(SRC)
windows-arm64:
	@echo "Building Windows arm64..."
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-arm64.exe $(SRC)

darwin: darwin-amd64 darwin-arm64

darwin-amd64:
	@echo "Building Darwin amd64..."
	GOOS=darwin GOARCH=amd64  go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64  $(SRC)
darwin-arm64:
	@echo "Building Darwin arm64..."
	GOOS=darwin GOARCH=arm64  go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64  $(SRC)

bin-clean:
	@echo "Cleaning..."
	rm -rf bin/

snap-build:
	@echo "Building Snap packages..."
	mkdir -p ./snap-builds
	snapcraft pack --use-lxd --output ./snap-builds
	@echo "Snap packages created in ./snap-builds/"

snap-clean:
	@echo "Cleaning Snap packages..."
	snapcraft clean --use-lxd
	rm -rf ./snap-builds/*

snap-install:
	@echo "Detecting OS and architecture with go env..."
	@OS=$$(go env GOOS); \
	ARCH=$$(go env GOARCH); \
	if [ "$$OS" != "linux" ]; then \
		echo "Snap installation is only supported on Linux"; \
		exit 1; \
	fi; \
	if [ "$$ARCH" != "amd64" ] && [ "$$ARCH" != "arm64" ]; then \
		echo "Unsupported architecture:   $$ARCH"; \
		exit 1; \
	fi; \
	SNAP_FILE=./snap-builds/$(APP_NAME)_$(VERSION)_$$ARCH.snap; \
	if [ -f "$$SNAP_FILE" ]; then \
		echo "Installing $$SNAP_FILE..."; \
		sudo snap install --dangerous "$$SNAP_FILE"; \
	else \
		echo "Snap file $$SNAP_FILE not found. Building snaps..."; \
 		$(MAKE) snap-build; \
		if [ -f "$$SNAP_FILE" ]; then \
			echo "Installing $$SNAP_FILE..."; \
			sudo snap install --dangerous "$$SNAP_FILE"; \
		else \
			echo "Failed to build or find snap for $$ARCH"; \
			exit 1; \
		fi; \
	fi
	@echo "Installation completed. Run with: snap run $(APP_NAME)"

snap-upload:
	@echo "Uploading Snap packages to Snap Store..."
	snapcraft upload --release=stable ./snap-builds/$(APP_NAME)_$(VERSION)_amd64.snap
	snapcraft upload --release=stable ./snap-builds/$(APP_NAME)_$(VERSION)_arm64.snap
	@echo "Uploads completed. Check status with: snapcraft status $(APP_NAME)"	

.PHONY: all build bin-clean
.PHONY: linux linux-amd64 linux-arm64
.PHONY: windows windows-amd64 windows-arm64
.PHONY: darwin darwin-amd64 darwin-arm64
.PHONY: snap-build snap-clean snap-install snap-upload

