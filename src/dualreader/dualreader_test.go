package dualreader

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
)

func TestDualReader(t *testing.T) {
	// Testdaten
	input := make([]byte, 1000*1000*1000) // 1GB

	_, err := rand.Read(input)
	if err != nil {
		t.Fatalf("Failed to generate random input: %v", err)
	}
	source := io.NopCloser(bytes.NewReader(input))

	// init daulreader
	dr := NewDualReader(source)
	r1, r2 := dr.Readers()

	// read both
	var buf1, buf2 bytes.Buffer
	done := make(chan struct{})

	go func() {
		_, err := io.Copy(&buf1, r1)
		if err != nil {
			t.Errorf("Reader 1 copy error: %v", err)
		}
		done <- struct{}{}
	}()

	go func() {
		_, err := io.Copy(&buf2, r2)
		if err != nil {
			t.Errorf("Reader 2 copy error: %v", err)
		}
		done <- struct{}{}
	}()

	// wait for both readers
	<-done
	<-done

	// compare
	if !bytes.Equal(buf1.Bytes(), input) {
		t.Errorf("Reader 1 output mismatch.\nExpected: %q\nGot: %q", input, buf1.String())
	}
	if !bytes.Equal(buf2.Bytes(), input) {
		t.Errorf("Reader 2 output mismatch.\nExpected: %q\nGot: %q", input, buf2.String())
	}
}
