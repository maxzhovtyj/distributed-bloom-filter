
proto-gen:
	protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative /Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/pkg/proto/bloom.proto

deploy:
	docker compose down
	docker build -t bloomnode .
	docker compose up -d

load-test:
	vegeta attack -targets=targets.txt -rate=5000/s -duration=60s -output=results.bin && cat results.bin | vegeta report