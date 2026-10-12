BINARY_NAME = confluence
BIN_DIR     = ./bin
CMD_DIR     = ./cmd/confluence

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     = -X github.com/atlassian/confluence-cli/internal/build.Version=$(VERSION)

.PHONY: all build test fmt lint tidy clean check-skills check-generated-skill sync-skills release-check

all: build

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

test:
	go test -v ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)

# Verify that agent-skill mirrors are up-to-date
check-skills:
	@bash scripts/check-generated-skill.sh

check-generated-skill:
	@bash scripts/check-generated-skill.sh

# Regenerate .claude/skills/confluence and .agents/skills/confluence from skills/confluence
sync-skills:
	@bash scripts/sync-skills.sh

release-check:
	@bash scripts/check-release-version.sh $(VERSION)
