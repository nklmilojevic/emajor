WORKFLOW_UUID := 6F1C9D83-A2B3-4E5F-8C7D-1E2F3A4B5C6D
ALFRED_PREFS  := $(HOME)/Library/Application Support/Alfred/Alfred.alfredpreferences
WORKFLOW_DIR  := $(ALFRED_PREFS)/workflows/user.workflow.$(WORKFLOW_UUID)

.PHONY: build build-universal install package clean

build:
	CGO_ENABLED=1 go build -o emajor .

build-universal:
	CGO_ENABLED=1 GOARCH=arm64 go build -o emajor_arm64 .
	CGO_ENABLED=1 GOARCH=amd64 SDKROOT=$(shell xcrun --sdk macosx --show-sdk-path) go build -o emajor_amd64 .
	lipo -create -output emajor emajor_arm64 emajor_amd64
	@rm emajor_arm64 emajor_amd64

install: build
	mkdir -p "$(WORKFLOW_DIR)"
	cp emajor "$(WORKFLOW_DIR)/"
	cp workflow/info.plist "$(WORKFLOW_DIR)/"
	cp workflow/icon.png "$(WORKFLOW_DIR)/"
	@echo "✓ Installed to Alfred as workflow $(WORKFLOW_UUID)"
	@echo "  Keyword: em <query>"

package: build-universal
	@rm -rf /tmp/emajor_pkg && mkdir -p /tmp/emajor_pkg
	cp emajor /tmp/emajor_pkg/
	cp workflow/info.plist /tmp/emajor_pkg/
	cp workflow/icon.png /tmp/emajor_pkg/
	cd /tmp/emajor_pkg && zip -r emajor.alfredworkflow . -x "*.DS_Store"
	mv /tmp/emajor_pkg/emajor.alfredworkflow .
	@rm -rf /tmp/emajor_pkg
	@echo "✓ Packaged emajor.alfredworkflow (double-click to install)"

clean:
	rm -f emajor emajor_arm64 emajor_amd64 emajor.alfredworkflow
