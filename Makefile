BINARY  := monk
INSTALL := $(HOME)/.local/bin

.PHONY: build install test clean

# Default: build the binary
build:
	cd src && go build -o ../$(BINARY) .

# Build and copy to ~/.local/bin
install: build
	@mkdir -p $(INSTALL)
	@cp $(BINARY) $(INSTALL)/$(BINARY)
	@echo "installed → $(INSTALL)/$(BINARY)"

# Run all tests
test:
	cd src && go test ./...

# Remove build artifacts
clean:
	@rm -f $(BINARY)
	@echo "clean"
