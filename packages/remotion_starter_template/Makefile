.PHONY: help dev build export_video clean

# Default target
help:
	@echo "Available commands:"
	@echo "  make dev          - Start Remotion Studio for development"
	@echo "  make build        - Build the Remotion project"
	@echo "  make export_video - Export video (requires CONFIG_JSON and optional OUTPUT)"
	@echo "  make clean        - Clean output directory and node_modules"
	@echo ""
	@echo "Usage examples:"
	@echo "  make export_video CONFIG_JSON='{\\"composition\\":...}' OUTPUT=my_video"
	@echo "  make export_video CONFIG_JSON='{\\"composition\\":...}'"

dev:
	npm run dev

build:
	npm run build

export_video:
ifndef CONFIG_JSON
	@echo "Error: CONFIG_JSON parameter is required"
	@echo "Usage: make export_video CONFIG_JSON='{\\"composition\\":{...},\\"elements\\":[...]}' [OUTPUT=filename]"
	@exit 1
endif
ifdef OUTPUT
	./scripts/export_video_full.sh '$(CONFIG_JSON)' "$(OUTPUT)"
else
	./scripts/export_video_full.sh '$(CONFIG_JSON)' "video_$$(date +%s)"
endif

clean:
	rm -rf output/
	rm -rf node_modules/
	@echo "Cleaned output directory and node_modules"