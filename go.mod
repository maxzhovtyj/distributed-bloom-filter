module github.com/maxzhovtyj/distributed-bloom-filter

go 1.25.3

replace github.com/bits-and-blooms/bloom/v3 => github.com/maxzhovtyj/bloom/v3 v3.0.0-20251026163504-289c255ad8a3

require (
	github.com/bits-and-blooms/bloom/v3 v3.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.23.2
	github.com/spaolacci/murmur3 v1.1.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)
