package bloomdata

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
)

func Read(input string, output chan<- []byte) error {
	defer close(output)

	f, err := os.Open(input)
	if err != nil {
		return err
	}

	reader := csv.NewReader(f)
	reader.Comma = '\t'

	for {
		read, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if len(read) != 2 {
			continue
		}

		uid := []byte(read[0])

		output <- uid
	}

	return nil
}
