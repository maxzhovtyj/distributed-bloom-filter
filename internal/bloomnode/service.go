package bloomnode

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

const bfdPath = "./bloom_filter.bfd"

var (
	testRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "test_requests_count",
		Help: "The total number of processed events",
	})
	testRequestLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "test_request_latency",
		Help: "The total latency of processed events",
	}, []string{"protocol"})

	insertRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "insert_requests_count",
		Help: "The total number of processed events",
	})
	insertRequestLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "insert_request_latency",
		Help: "The total latency of processed events",
	})
)

type Service struct {
	masterURI string

	bloomFilter   atomic.Pointer[bloom.BloomFilter]
	elementsCount atomic.Uint64

	bloomproto.UnimplementedDistributedBloomFilterServer
}

func NewService(masterURI string) *Service {
	s := &Service{
		masterURI: masterURI,
	}

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "bloom_filter_size",
		Help: "The size of bloom filter in bytes",
	}, func() float64 {
		return float64(s.elementsCount.Load())
	})

	return s
}

func (s *Service) Init() error {
	err := s.loadFromDisk()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	go s.RunSyncWorker()

	return nil
}

func (s *Service) TestHTTP(w http.ResponseWriter, r *http.Request) {
	testRequests.Inc()
	start := time.Now()
	defer func() {
		testRequestLatency.WithLabelValues("http").Observe(time.Since(start).Seconds())
	}()

	uid := []byte(r.URL.Query().Get("uid"))

	res, err := s.TestBloomFilter(uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}

	_, _ = w.Write([]byte("OK"))
}

func (s *Service) TestBloomFilter(uid []byte) (bool, error) {
	bfd := s.bloomFilter.Load()
	if bfd == nil {
		return false, fmt.Errorf("bloom filter not found")
	}

	return bfd.Test(uid), nil
}

func (s *Service) Test(ctx context.Context, request *bloomproto.TestRequest) (*bloomproto.TestResponse, error) {
	testRequests.Inc()
	start := time.Now()
	defer func() {
		testRequestLatency.WithLabelValues("grpc").Observe(time.Since(start).Seconds())
	}()

	res, err := s.TestBloomFilter(request.Key)
	if err != nil {
		return &bloomproto.TestResponse{}, err
	}

	return &bloomproto.TestResponse{
		IsPresent: res,
	}, nil
}

func (s *Service) Insert(req grpc.ClientStreamingServer[bloomproto.InsertRequest, bloomproto.InsertResponse]) error {
	insertRequests.Inc()
	start := time.Now()
	defer func() {
		insertRequestLatency.Observe(time.Since(start).Seconds())
	}()

	bfd := bloom.NewWithEstimates(uint(s.elementsCount.Load()), 0.001)

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
					break
				}

				break
			}

			return err
		}

		bfd.Add(recv.Key)
	}

	s.bloomFilter.Store(bfd)

	go func() {
		err := s.storeToDisk()
		if err != nil {
			log.Printf("Error storing bloom filter: %v", err)
			return
		}

		log.Println("Successfully updated and saved bloom filter to disk")
	}()

	log.Println("Successfully updated bloom filter")

	return nil
}

func (s *Service) loadFromDisk() error {
	raw, err := os.ReadFile(bfdPath)
	if err != nil {
		return err
	}

	var bfd bloom.BloomFilter

	err = bfd.UnmarshalBinary(raw)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) storeToDisk() error {
	bfd := s.bloomFilter.Load()

	raw, err := bfd.MarshalBinary()
	if err != nil {
		return err
	}

	err = os.WriteFile(bfdPath, raw, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) PrepareBloomFilter(
	ctx context.Context,
	req *bloomproto.PrepareBloomFilterRequest,
) (*bloomproto.PrepareBloomFilterResponse, error) {
	s.elementsCount.Store(uint64(req.ElementsCount))

	return &bloomproto.PrepareBloomFilterResponse{}, nil
}

func (s *Service) SyncBloomFilter(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	buf := bufio.NewWriter(w)

	n, err := s.bloomFilter.Load().WriteTo(buf)
	if err != nil {
		log.Println("Failed to write bloom filter for sync", err)
		return
	}

	log.Printf("Successfully synced in %s wrote: %d\n", time.Since(start), n)
}

func (s *Service) RunSyncWorker() {
	for {
		s.doSyncBloomFilter()
		time.Sleep(15 * time.Minute)
	}
}

func (s *Service) doSyncBloomFilter() {
	if s.masterURI == "" {
		return
	}

	start := time.Now()
	log.Println("Start syncing bloom filter")

	uri := fmt.Sprintf("%s/sync", s.masterURI)

	resp, err := http.Get(uri)
	if err != nil {
		log.Println("Failed to do http request for sync", err)
		return
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Println("Failed to close bloom filter for sync", err)
			return
		}
	}()

	var bfd bloom.BloomFilter

	n, err := bfd.ReadFrom(resp.Body)
	if err != nil {
		log.Println("Failed to read bloom filter for sync", err)
		return
	}

	s.bloomFilter.Store(&bfd)

	log.Printf("Bloom filter syncked in %s read: %d\n", time.Since(start), n)
}
