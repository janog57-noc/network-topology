.PHONY: help generate clean serve

OUT_DIR=./out

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

generate: ## Generate topology HTML from NetBox
	bunx netbox-to-shumoku --format html --output $(OUT_DIR)/index --legend

clean: ## Clean build artifacts
	rm -rf $(OUT_DIR)

serve: generate ## Generate and serve locally
	cd $(OUT_DIR) && python3 -m http.server 8000
