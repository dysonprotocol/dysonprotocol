# Makefile

# Default to Alice's address if none is set in the environment
ADDRESS ?= $(shell dysond keys show alice --output json | jq -r .address)

.PHONY: watch handle-change update-script upload-file create-script update

watch:
	@find script.py storage -type f | \
	entr -dp -s 'make handle-change FILE="$$0"'

handle-change:
	@echo "File changed: $(FILE)"
ifneq ($(FILE),)

	@if [ "$(FILE)" = `pwd`/"script.py" ]; then \
		$(MAKE) update-script; \
	else \
		$(MAKE) upload-file FILE="$(FILE)"; \
	fi
else
	@echo "No FILE specified, ignoring change."
endif

update-script:
	@echo "Updating script at address: $(ADDRESS)"
	@dysond tx script update \
		--from "$(ADDRESS)" \
		--gas-adjustment 2 \
		--gas auto \
		-y -o json \
		--code-path script.py \
	| jq -r .txhash \
	| xargs dysond q wait-tx -o json \
	| jq

upload-file:
	@index=$$(echo "$$FILE" | sed 's|.*storage/||') && \
	echo "Uploading file '$$index'..."; \
	dysond tx storage set \
		--gas-adjustment 2 \
		--gas auto \
		--from "$(ADDRESS)" \
		--index "$$index" \
		--data "$$(cat "$$FILE")" \
		-y -o json \
	| jq -r .txhash \
	| xargs dysond q wait-tx -o json \
	| jq

update-all:
	@echo "Updating script and uploading all storage files..."
	@$(MAKE) update-script
	@echo "Uploading all storage files..."
	@find storage -type f | while read -r file; do \
		$(MAKE) upload-file FILE="$$file"; \
	done
	@echo "Update complete!"