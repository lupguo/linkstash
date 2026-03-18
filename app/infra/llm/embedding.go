package llm

import (
	"encoding/binary"
	"math"
)

// Float32sToBytes converts a slice of float32 values to a byte slice using LittleEndian encoding.
// This is useful for storing embedding vectors as BLOBs in a database.
func Float32sToBytes(floats []float32) []byte {
	buf := make([]byte, len(floats)*4)
	for i, f := range floats {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// BytesToFloat32s converts a byte slice back to a slice of float32 values using LittleEndian encoding.
func BytesToFloat32s(data []byte) []float32 {
	count := len(data) / 4
	floats := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4:])
		floats[i] = math.Float32frombits(bits)
	}
	return floats
}

// CosineSimilarity computes the cosine similarity between two float32 vectors.
// Returns 0 if either vector has zero length.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dotProduct += ai * bi
		normA += ai * ai
		normB += bi * bi
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
