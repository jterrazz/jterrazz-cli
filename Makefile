.PHONY: build install uninstall check test clean help

# ==============================================================================
# Configuration
# ==============================================================================

BINARY          := j
JTERRAZZ_DIR    := $(HOME)/.jterrazz
BIN_DIR         := $(JTERRAZZ_DIR)/bin
INSTALL_PATH    := $(BIN_DIR)/$(BINARY)
ZSHRC_SOURCE    := dotfiles/applications/zsh/zshrc.sh
OLD_CONFIG_DIR  := $(HOME)/.config/jterrazz
OLD_INSTALL     := /usr/local/bin/$(BINARY)

CYAN  := \033[36m
GREEN := \033[32m
DIM   := \033[2m
RESET := \033[0m

# ==============================================================================
# Targets
# ==============================================================================

help: ## Show available targets
	@printf "$(CYAN)jterrazz-cli$(RESET)\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-12s$(RESET) %s\n", $$1, $$2}'

build: ## Build the binary
	@go build -o $(BINARY) ./src/cmd/j
	@printf "$(GREEN)✓$(RESET) Built ./$(BINARY)\n"

install: build ## Build and install to ~/.jterrazz/bin
	@mkdir -p $(BIN_DIR)
	@cp $(BINARY) $(INSTALL_PATH)
	@chmod +x $(INSTALL_PATH)
	@rm $(BINARY)
	@printf "$(GREEN)✓$(RESET) Installed $(INSTALL_PATH)\n"
	@if [ -f "$(OLD_INSTALL)" ]; then \
		sudo rm "$(OLD_INSTALL)"; \
		printf "$(GREEN)✓$(RESET) Removed old $(OLD_INSTALL)\n"; \
	fi
	@if [ -f "$(OLD_CONFIG_DIR)/jrc.json" ] && [ ! -f "$(JTERRAZZ_DIR)/config.json" ]; then \
		cp "$(OLD_CONFIG_DIR)/jrc.json" "$(JTERRAZZ_DIR)/config.json"; \
		printf "$(GREEN)✓$(RESET) Migrated config.json\n"; \
	fi
	@if [ -d "$(OLD_CONFIG_DIR)/tailscale" ] && [ ! -d "$(JTERRAZZ_DIR)/tailscale" ]; then \
		cp -r "$(OLD_CONFIG_DIR)/tailscale" "$(JTERRAZZ_DIR)/tailscale"; \
		printf "$(GREEN)✓$(RESET) Migrated tailscale state\n"; \
	fi
	@if [ -f "$$HOME/.zshrc" ]; then \
		if ! grep -q "$(ZSHRC_SOURCE)" "$$HOME/.zshrc"; then \
			printf '\n# jterrazz-cli\nsource $(PWD)/$(ZSHRC_SOURCE)\n' >> "$$HOME/.zshrc"; \
			printf "$(GREEN)✓$(RESET) Added source to ~/.zshrc\n"; \
		fi \
	else \
		printf "$(DIM)~/.zshrc not found — add manually: source $(PWD)/$(ZSHRC_SOURCE)$(RESET)\n"; \
	fi
	@printf "$(GREEN)✓$(RESET) Done — run $(DIM)source ~/.zshrc$(RESET) then $(DIM)j help$(RESET)\n"

uninstall: ## Remove binary from ~/.jterrazz/bin
	@if [ -f "$(INSTALL_PATH)" ]; then \
		rm "$(INSTALL_PATH)"; \
		printf "$(GREEN)✓$(RESET) Removed $(INSTALL_PATH)\n"; \
	else \
		printf "$(DIM)$(BINARY) not found at $(INSTALL_PATH)$(RESET)\n"; \
	fi
	@if [ -f "$(OLD_INSTALL)" ]; then \
		sudo rm "$(OLD_INSTALL)"; \
		printf "$(GREEN)✓$(RESET) Removed $(OLD_INSTALL)\n"; \
	fi

check: ## Verify installation
	@if command -v $(BINARY) >/dev/null 2>&1; then \
		printf "$(GREEN)✓$(RESET) $(BINARY) $(DIM)$$(which $(BINARY))$(RESET)\n"; \
	else \
		printf "✗ $(BINARY) not found in PATH\n"; \
	fi
	@if [ -d "$(JTERRAZZ_DIR)" ]; then \
		printf "$(GREEN)✓$(RESET) ~/.jterrazz $(DIM)exists$(RESET)\n"; \
	else \
		printf "✗ ~/.jterrazz not found\n"; \
	fi

test: ## Run tests
	@go test ./src/...

clean: ## Remove build artifacts
	@rm -f $(BINARY)
	@printf "$(GREEN)✓$(RESET) Cleaned\n"
