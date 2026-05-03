SHELL := /bin/bash

mock_gen:
	@echo -e "=> Mockery:"
	@mockery && \
		echo -e "==> Mockery - Ok" || (echo -e "==> Mockery - Failed" && exit 1)

run_test:
	@echo -e "=> Run tests:"
	@go test ./... && \
		echo -e "==> Tests - Ok" || (echo -e "==> Tests - Failed" && exit 1)

check: mock_gen run_test