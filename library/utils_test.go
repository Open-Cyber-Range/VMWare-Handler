package library

import (
	"math"
	"testing"
)

func TestScalingBytes(t *testing.T) {
	t.Parallel()
	if ScaleBytesByUnit(uint64(math.Pow(1024, 2))) != 1 {
		t.Fatalf("Failed to scale bytes by unit")
	}
}

func TestGeneratingStringWithCorrectLength(t *testing.T) {
	t.Parallel()
	length := 3
	if len(CreateRandomString(length)) != length {
		t.Fatalf("Failed to generate string with correct length")
	}
}
