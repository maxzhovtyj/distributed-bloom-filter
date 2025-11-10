package main

import (
	"encoding/json"
	"fmt"
	dbf "github.com/maxzhovtyj/distributed-bloom-filter/internal/bloom"
	"net/http"
	"runtime"
)

func main() {
	mux := http.NewServeMux()

	dbf.Init()
	dbf.InitMonolith()

	mux.HandleFunc("/mem-stats", func(w http.ResponseWriter, r *http.Request) {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		w.Header().Set("Content-Type", "application/json")
		raw, err := json.Marshal(mem)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(raw)
	})

	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		uid := []byte(r.URL.Query().Get("uid"))

		res := dbf.TestMonolith(uid)

		_, _ = w.Write([]byte(fmt.Sprintf(
			"UID: %s\nPresent: %v",
			uid, res,
		)))
	})

	mux.HandleFunc("/distributed/bfd", func(w http.ResponseWriter, r *http.Request) {
		dbf.Collect()
	})

	mux.HandleFunc("/distributed/test", func(w http.ResponseWriter, r *http.Request) {
		uid := []byte(r.URL.Query().Get("uid"))

		res, nodeID, err := dbf.Test(uid)

		_, _ = w.Write([]byte(fmt.Sprintf(
			"UID: %s\nPresent: %v\nNode: %s\nError: %v",
			uid, res, nodeID, err,
		)))
	})

	err := http.ListenAndServe(":8000", mux)
	if err != nil {
		return
	}
}
