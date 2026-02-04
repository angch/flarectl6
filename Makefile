.PHONY: fetch-ref clean-ref

fetch-ref:
	mkdir -p ref
	# Fetch legacy flarectl (v0.116.0)
	@if [ ! -d "ref/flarectl" ]; then \
		echo "Fetching legacy flarectl..."; \
		git clone --branch v0.116.0 --depth 1 https://github.com/cloudflare/cloudflare-go ref/temp_legacy; \
		mv ref/temp_legacy/cmd/flarectl ref/flarectl; \
		rm -rf ref/temp_legacy; \
	else \
		echo "ref/flarectl already exists"; \
	fi
	# Fetch cloudflare-go v6 (v6.6.0)
	@if [ ! -d "ref/cloudflare-go" ]; then \
		echo "Fetching cloudflare-go v6..."; \
		git clone --branch v6.6.0 --depth 1 https://github.com/cloudflare/cloudflare-go ref/cloudflare-go; \
	else \
		echo "ref/cloudflare-go already exists"; \
	fi

clean-ref:
	rm -rf ref
