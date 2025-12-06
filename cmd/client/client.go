package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"github.com/google/uuid"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/sdk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"unsafe"
)

var (
	falsePositiveResponse = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sdk_false_positive_response_count",
		Help: "The total number of false positive responses",
	})
	falseNegativeResponse = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sdk_false_negative_response_count",
		Help: "The total number of false negative responses",
	})
	testRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sdk_test_requests_count",
		Help: "The total number of present responses",
	})
	testRequestsErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sdk_test_requests_errors_count",
		Help: "The total number of errors present responses",
	})
)

type LoadTestWorkerPool struct {
	sdk     *sdk.DistributedBloomFilter
	workers int
	tasks   chan Task
}

type Task struct {
	Key     string
	IsInSet bool
}

func (p *LoadTestWorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		go p.runWorker()
	}
}

func (p *LoadTestWorkerPool) runWorker() {
	for element := range p.tasks {
		ptr := unsafe.StringData(element.Key)
		byteSlice := unsafe.Slice(ptr, len(element.Key))

		isPresent, err := p.sdk.Test(byteSlice)
		if err != nil {
			testRequestsErrors.Inc()
			continue
		}

		if isPresent && !element.IsInSet {
			falsePositiveResponse.Inc()
		}

		if !isPresent && element.IsInSet {
			falseNegativeResponse.Inc()
		}

		testRequests.Inc()
	}
}

var (
	randSlots = [10]bool{false, false, false, false, false, false, false, false, false, true}
	randIdx   atomic.Uint64
)

var (
	zookeeperURI = flag.String("zookeeper", "localhost:7000", "Zookeeper URI")
	path         = flag.String("path", "/root/distributed-bloom-filter/uuids_100000000.csv", "Zookeeper path")
)

func main() {
	flag.Parse()

	bloomSDK := sdk.NewDistributedBloomFilter(*zookeeperURI)

	err := bloomSDK.Init()
	if err != nil {
		log.Panicf("Failed to init sdk: %v", err)
	}

	f, err := os.Open(*path)
	if err != nil {
		log.Panicf("Failed to open file: %v", err)
	}

	go runSDKHTTPHandler()

	wp := LoadTestWorkerPool{
		sdk:     bloomSDK,
		workers: 100,
		tasks:   make(chan Task, 100_000),
	}

	wp.Start()

	defer close(wp.tasks)

	r := csv.NewReader(f)

	for {
		read, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			log.Panicf("Failed to read line: %v", err)
		}

		t := Task{
			Key:     read[0],
			IsInSet: true,
		}

		isRandom := randSlots[randIdx.Add(1)%uint64(len(randSlots))]
		if isRandom {
			t.Key = uuid.New().String()
			t.IsInSet = false
		}

		wp.tasks <- t
	}

	log.Println("Finished")

	select {}
}

func runSDKHTTPHandler() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Println("Start serving http on :9000")
	if err := http.ListenAndServe(":9000", mux); err != nil {
		panic(err)
	}
}
