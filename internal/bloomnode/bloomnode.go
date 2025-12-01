package bloomnode

import (
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
)

func Run(masterNodeURI string) {
	tcpSocket, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()

	service := NewService()

	err = service.Init()
	if err != nil {
		log.Panicf("Error initializing service: %v", err)
	}

	bloomproto.RegisterDistributedBloomFilterServer(grpcServer, service)

	go func() {
		log.Println("Start listening grpcServer on :8000")

		err = grpcServer.Serve(tcpSocket)
		if err != nil {
			panic(err)
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/test", service.TestHTTP)
	mux.HandleFunc("/sync", service.SyncBloomFilter)

	log.Println("Start serving http on :9000")
	if httpErr := http.ListenAndServe(":9000", mux); httpErr != nil {
		panic(httpErr)
	}
}
