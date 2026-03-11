APP_NAME := mocky
BUILD_DIR := build
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
GOCACHE ?= $(CURDIR)/.gocache

export GOCACHE

.PHONY: build install install-local uninstall clean

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/mocky

install:
	install -d $(BINDIR)
	install $(BUILD_DIR)/$(APP_NAME) $(BINDIR)/$(APP_NAME)

install-local:
	go install ./cmd/mocky

uninstall:
	rm -f $(BINDIR)/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR)
