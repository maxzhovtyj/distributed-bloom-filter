module github.com/maxzhovtyj/distributed-bloom-filter

go 1.25.3

replace github.com/bits-and-blooms/bloom/v3 => github.com/maxzhovtyj/bloom/v3 v3.0.0-20251026163504-289c255ad8a3

require (
	github.com/bits-and-blooms/bloom/v3 v3.0.0-00010101000000-000000000000
	github.com/spaolacci/murmur3 v1.1.0
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/bits-and-blooms/bitset v1.24.3 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
)
