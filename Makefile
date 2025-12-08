
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

sdk:
	GOOS=linux GOARCH=amd64 go build -o bin/sdk-linux-amd64 ./cmd/client

build-bloom-node:
	GOOS=linux GOARCH=amd64 go build -o bin/node-linux-amd64 ./cmd/bloomnode

uuid-generate:
	GOOS=linux GOARCH=amd64 go build -o bin/uuids-linux-amd64 ./cmd/uuid-generator

deploy-bloom-%: build-bloom-node
	rsync -avz -e "ssh -i ~/.ssh/digitalocean -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" --progress --rsync-path="sudo rsync" \
	  	bin/node-linux-amd64 \
		root@$(HOST):/root/distributed-bloom-filter/bin/
	echo "pm2 stop bloom-node;pm2 start /root/distributed-bloom-filter/bin/node-linux-amd64 --name=\"bloom-node\" && exit" | ssh -i ~/.ssh/digitalocean root@$(HOST) /bin/sh

deploy-bloom-fr1: HOST = 161.35.74.38
deploy-bloom-fr2: HOST = 134.209.244.237
deploy-bloom-fr3: HOST = 46.101.123.0
deploy-bloom-fr4: HOST = 64.225.105.207

CONFIG_PATH = /root/distributed-bloom-filter/bin/config.yml

deploy-zookeeper: zookeeper
	rsync -avz -e "ssh -i ~/.ssh/digitalocean -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" --progress --rsync-path="sudo rsync" \
		cmd/zookeeper/config.yml \
		bin/zookeeper-linux-amd64 \
		root@64.225.105.207:/root/distributed-bloom-filter/bin/
	echo "cd /root/distributed-bloom-filter/bin;pm2 stop zookeeper;pm2 start zookeeper-linux-amd64 --name=\"zookeeper\" -- -config=$(CONFIG_PATH) && exit" | ssh -i ~/.ssh/digitalocean root@64.225.105.207 /bin/sh

deploy-sdk: sdk
	rsync -avz -e "ssh -i ~/.ssh/digitalocean -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" --progress --rsync-path="sudo rsync" \
		bin/sdk-linux-amd64 \
		root@64.225.105.207:/root/distributed-bloom-filter/bin/
	echo "cd /root/distributed-bloom-filter/bin;pm2 stop client-sdk;pm2 start sdk-linux-amd64 --name=\"client-sdk\" && exit" | ssh -i ~/.ssh/digitalocean root@64.225.105.207 /bin/sh
