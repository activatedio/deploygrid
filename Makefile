
generate_mocks:
	find . -type f | grep mock_ | xargs rm || true
	go install github.com/vektra/mockery/v2@v2.46.3
	mockery

dev_kind:
	./kind/teardown.sh || true
	./kind/setup.sh

test:
	go test ./...

serve:
	REPOSITORY_COMMON_MODE=stub LOGGING_DEV_MODE=true go run ./cmd/main
