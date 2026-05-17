# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2

GITFLOWER_BIN ?= $(shell pwd)/gitflower
CASTS_DIR     ?= $(shell pwd)/test/e2e/casts
# Optional build tags. Default off; enable per-feature with e.g.
#   make build BUILDTAGS="with_review_merge with_mrs"
BUILDTAGS     ?=
GO_BUILD_TAGS := $(if $(BUILDTAGS),-tags "$(BUILDTAGS)",)

.PHONY: build test test-all e2e e2e-format e2e-record lint clean help

build:
	go build $(GO_BUILD_TAGS) -o $(GITFLOWER_BIN) .

# `go test ./...` covers:
#   - review/: format Render/Parse round-trip
#   - tui/:    bubbletea Model driven via synthetic KeyPressMsgs
#   - test/e2e/: real binary spawned under a PTY against the fixture repo,
#                walked through a scripted keystroke scenario, then the
#                produced .review is asserted.
test:
	go test $(GO_BUILD_TAGS) ./...

# Same as `test`, but with every build-tagged feature turned on so
# the gated subcommand tests run too.
test-all:
	go test -tags "with_review_merge with_mrs" ./...

# Format-pipeline smoke check: builds the binary, rebuilds the fixture
# repo via setup.sh, runs `gitflower review --no-tui`, then diffs the
# resulting .review against the golden after normalising dates/SHAs.
e2e-format: build
	./test/e2e/smoke.sh

# Re-record asciicast files for every PTY-driven e2e test so you can
# replay what the test actually does in your terminal:
#
#   make e2e-record
#   asciinema play test/e2e/casts/TestReviewViaPTY.cast
#
# Renders to a GIF via `agg`:
#   agg test/e2e/casts/TestReviewViaPTY.cast > demo.gif
#
# Casts have the same width/height the test gave the PTY (120x40),
# so terminal sizing and colours match what the test saw.
e2e-record:
	@mkdir -p $(CASTS_DIR)
	@echo "recording PTY tests → $(CASTS_DIR)/<TestName>.cast"
	ASCIINEMA_OUT_DIR=$(CASTS_DIR) go test $(GO_BUILD_TAGS) ./test/e2e/ -count=1 -run 'TestReview.*PTY|TestReviewRestart'
	@echo
	@echo 'Play with:  asciinema play $(CASTS_DIR)/<TestName>.cast'
	@ls -la $(CASTS_DIR)

# Convenience: run every test layer in one go.
e2e: test e2e-format

lint:
	go vet ./...

clean:
	rm -f $(GITFLOWER_BIN)
	rm -rf /tmp/gitflower-e2e-repo
	rm -rf $(CASTS_DIR)

help:
	@echo 'Targets:'
	@echo '  build           compile the gitflower binary'
	@echo '                  (BUILDTAGS="with_review_merge with_mrs" to enable gated subcommands)'
	@echo '  test            run all Go tests (default tags only)'
	@echo '  test-all        run tests with every with_* tag enabled'
	@echo '  e2e-format      format pipeline smoke check (build + scaffold + golden diff)'
	@echo '  e2e-record      re-record PTY tests as asciicast files in $(CASTS_DIR)'
	@echo '  e2e             everything above'
	@echo '  lint            go vet'
	@echo '  clean           remove the binary, the test repo, and recorded casts'
