package bloomnode

import (
	"context"
	"errors"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"time"
)

var (
	testRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "test_requests_count",
		Help: "The total number of processed events",
	})
	testRequestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "test_request_latency",
		Help: "The total latency of processed events",
	}, []string{"protocol"})
)

type Service struct {
	bloomFilter *bloom.BloomFilter

	bloomproto.UnimplementedDistributedBloomFilterServer
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) TestHTTP(w http.ResponseWriter, r *http.Request) {
	testRequests.Inc()
	start := time.Now()
	defer func() {
		testRequestLatency.WithLabelValues("http").Observe(time.Since(start).Seconds())
	}()

	uid := []byte(r.URL.Query().Get("uid"))

	if s.TestBloomFilter(uid) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}

	_, _ = w.Write([]byte("OK"))
}

func (s *Service) TestBloomFilter(uid []byte) bool {
	return s.bloomFilter.Test(uid)
}

func (s *Service) Test(ctx context.Context, request *bloomproto.TestRequest) (*bloomproto.TestResponse, error) {
	if s.bloomFilter == nil {
		return nil, status.Error(codes.FailedPrecondition, "bloom filter is not initialized")
	}

	testRequests.Inc()
	start := time.Now()
	defer func() {
		testRequestLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	return &bloomproto.TestResponse{
		IsPresent: s.TestBloomFilter(request.Key),
	}, nil
}

func (s *Service) Insert(req grpc.ClientStreamingServer[bloomproto.InsertRequest, bloomproto.InsertResponse]) error {
	if s.bloomFilter == nil {
		return status.Error(codes.Internal, "BloomFilter is nil")
	}

	for {
		select {
		case <-req.Context().Done():
			return req.Context().Err()
		default:
			// ok
		}

		recv, err := req.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = req.SendAndClose(&bloomproto.InsertResponse{})
				if err != nil {
					return err
				}
			}

			return err
		}

		s.bloomFilter.Add(recv.Key)
	}
}

func (s *Service) PrepareBloomFilter(
	ctx context.Context,
	req *bloomproto.PrepareBloomFilterRequest,
) (*bloomproto.PrepareBloomFilterResponse, error) {
	s.bloomFilter = bloom.NewWithEstimates(uint(req.ElementsCount), 0.001)

	return &bloomproto.PrepareBloomFilterResponse{}, nil
}
