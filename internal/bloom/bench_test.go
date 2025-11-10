package dbf

import "testing"

func BenchmarkMonolith(b *testing.B) {
	InitMonolith()

	input := []byte("a8546a9d-2ad7-4de1-b4b6-de127b9aa984")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bfd.Test(input)
	}
}

func BenchmarkDistributed(b *testing.B) {
	Init()
	Collect()

	input := []byte("a8546a9d-2ad7-4de1-b4b6-de127b9aa984")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := s.ring.GetNode(input)
		node.BFD.Test(input)
	}
}
