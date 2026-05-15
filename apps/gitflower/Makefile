# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2

GITFLOWER_BIN ?= $(shell pwd)/gitflower

.PHONY: build test e2e e2e-vhs lint clean

build:
	go build -o $(GITFLOWER_BIN) .

test:
	go test ./...

e2e: build
	./test/e2e/smoke.sh

e2e-vhs: build
	./test/e2e/run.sh

lint:
	go vet ./...

clean:
	rm -f $(GITFLOWER_BIN)
	rm -rf /tmp/gitflower-e2e-repo

help:
	@echo 'Targets:'
	@echo '  build     compile the gitflower binary'
	@echo '  test      run unit + TUI-integration tests (no PTY needed)'
	@echo '  e2e       smoke check: build + scaffold + diff against golden'
	@echo '  e2e-vhs   VHS-driven scenarios (requires vhs + ttyd on PATH)'
	@echo '  lint      go vet'
	@echo '  clean     remove the binary and the test repo'
