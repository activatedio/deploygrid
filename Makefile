dev_kind:
	./kind/teardown.sh || true
	./kind/setup.sh

test:
	go test ./...

serve:
	LOGGING_DEV_MODE=true go run ./cmd/main
