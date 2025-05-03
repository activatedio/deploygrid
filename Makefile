dev_kind:
	kind delete cluster
	kind create cluster --config=./kind/kind-cluster.yaml
	kubectl --context kind-kind apply -f ./kind/cert-manager-crds.yaml
	kubectl --context kind-kind apply -f ./kind/bluebird-operator-crds.yaml
	kubectl --context kind-kind apply -f ./kind/helm-operator-crds.yaml
	kubectl --context kind-kind apply -f ./kind/test-resources.yaml

dev_containers:
	docker compose stop
	docker compose rm -f
	docker compose up -d

test:
	go test ./...

serve:
	go run ./cmd/main
