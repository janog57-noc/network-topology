.PHONY: help generate clean serve

OUT_DIR=./out

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

generate: ## Generate topology HTML from NetBox
	node src/ocx-integration/cli.mjs -o ocx-topology.json
	npx netbox-to-shumoku --format json --output netbox.json
	node src/merge-json.js netbox.json ocx-topology.json -o merged.json
	npx shumoku render merged.json --format html --output $(OUT_DIR)/index.html

clean: ## Clean build artifacts
	rm -rf $(OUT_DIR)

serve: generate ## Generate and serve locally
	cd $(OUT_DIR) && python3 -m http.server 8000
