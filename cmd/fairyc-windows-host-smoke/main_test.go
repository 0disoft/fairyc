package main

import "testing"

func TestBoundedBufferStoresPrefixWithoutShortWrite(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 5}

	written, err := buffer.Write([]byte("Fairy C"))
	if err != nil {
		t.Fatal(err)
	}
	if written != len("Fairy C") {
		t.Fatalf("Write() = %d; want %d", written, len("Fairy C"))
	}
	if buffer.Len() != 5 || buffer.buffer.String() != "Fairy" || !buffer.truncated {
		t.Fatalf("buffer = len %d, raw %q, truncated %t", buffer.Len(), buffer.buffer.String(), buffer.truncated)
	}
	if buffer.String() != "Fairy\n[output truncated]" {
		t.Fatalf("String() = %q", buffer.String())
	}
}

func TestBoundedBufferAcceptsExactLimit(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 5}

	written, err := buffer.Write([]byte("Fairy"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 5 || buffer.Len() != 5 || buffer.String() != "Fairy" || buffer.truncated {
		t.Fatalf("buffer = written %d, len %d, value %q, truncated %t", written, buffer.Len(), buffer.String(), buffer.truncated)
	}
}
