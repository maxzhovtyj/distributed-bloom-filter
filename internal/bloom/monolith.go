package dbf

import (
	"encoding/csv"
	"errors"
	"github.com/bits-and-blooms/bloom/v3"
	"io"
	"os"
)

var bfd *bloom.BloomFilter

func InitMonolith() {
	f, err := os.Open("/Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/data/idfa1.csv")
	if err != nil {
		panic(err)
	}

	bfd = bloom.NewWithEstimates(100_100, 0.01)

	reader := csv.NewReader(f)
	reader.Comma = '\t'

	for {
		read, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			panic(err)
		}

		if len(read) != 2 {
			continue
		}

		uid := []byte(read[0])
		bfd.Add(uid)
	}
}

func TestMonolith(key []byte) bool {
	return bfd.Test(key)
}
