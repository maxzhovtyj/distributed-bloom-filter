package bloomnode

import (
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
)

func Run() {
	tcpSocket, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()

	service := NewService()

	bloomproto.RegisterDistributedBloomFilterServer(grpcServer, service)

	go func() {
		log.Println("Start listening grpcServer on :8000")

		err = grpcServer.Serve(tcpSocket)
		if err != nil {
			panic(err)
		}
	}()

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/test", service.TestHTTP)

	log.Println("Start serving http on :9000")
	if httpErr := http.ListenAndServe(":9000", mux); httpErr != nil {
		panic(httpErr)
	}
}
