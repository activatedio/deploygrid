
generate_mocks:
	find . -type f | grep mock_ | xargs rm || true
	go install github.com/vektra/mockery/v2@v2.46.3
	mockery

dev_containers:
	docker compose stop
	docker compose rm -f
	docker compose up -d

dev_kind:
	./kind/teardown.sh || true
	./kind/setup.sh

test:
	go test ./...

serve:
	SWAGGER_SWAGGER_UI_URL=http://127.0.0.1:8081 REPOSITORY_COMMON_MODE=stub LOGGING_DEV_MODE=true go run ./cmd/main
