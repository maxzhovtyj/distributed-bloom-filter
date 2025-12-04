package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"log"
	"os"
	"time"
)

var elements = flag.Int("elements", 1000, "Number of elements to generate")

func main() {
	flag.Parse()

	log.Printf("Generating %d UUIDs...", *elements)
	start := time.Now()
	defer func() {
		log.Printf("Finished in %s", time.Since(start))
	}()

	filename := fmt.Sprintf("./uuids_%d_%d.csv", time.Now().Unix(), *elements)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	defer func() {
		_ = f.Close()
	}()

	writer := bufio.NewWriter(f)
	for range *elements {
		_, _ = writer.WriteString(uuid.New().String())
		_, _ = writer.WriteString("\n")
	}
}
