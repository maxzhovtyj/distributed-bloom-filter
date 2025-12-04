
proto-gen:
	protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative /Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/pkg/proto/bloom.proto

deploy:
	docker compose down
	docker build -t bloomnode .
	docker compose up -d

load-test:
	vegeta attack -targets=targets.txt -rate=5000/s -duration=60s -output=results.bin && cat results.bin | vegeta report

zookeeper:
	GOOS=linux GOARCH=amd64 go build -o bin/zookeeper-linux-amd64 ./cmd/zookeeper

build-bloom-node:
	GOOS=linux GOARCH=amd64 go build -o bin/node-linux-amd64 ./cmd/bloomnode

uuid-generate:
	GOOS=linux GOARCH=amd64 go build -o bin/uuids-linux-amd64 ./cmd/uuid-generator