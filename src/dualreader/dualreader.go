package dualreader

import (
	"io"
	"log"
)

type DualReader struct {
	r1 io.Reader
	r2 io.Reader
}

func NewDualReader(source io.ReadCloser) *DualReader {
	pr1, pw1 := io.Pipe()
	pr2, pw2 := io.Pipe()

	go func() {
		defer pw1.Close()
		defer pw2.Close()
		defer source.Close()

		multi := io.MultiWriter(pw1, pw2)
		_, err := io.Copy(multi, source)
		if err != nil {
			log.Println("[DualReader] error copying:", err)
		}
	}()

	return &DualReader{
		r1: pr1,
		r2: pr2,
	}
}

func (d *DualReader) Readers() (io.Reader, io.Reader) {
	return d.r1, d.r2
}
