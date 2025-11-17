package main

import (
	"fmt"
	"io"
	"iter"
)

// ——————————————————————————————————————————————————————————————————————
// IXMLBookBytes implementation (singleton-style per-book byte arrays)
// ——————————————————————————————————————————————————————————————————————
type xmlBookBytes struct {
	bufs map[string]*[]byte // book_id → *[]byte (shared reference)
}

func newXMLBookBytes() IXMLBookBytes {
	return &xmlBookBytes{
		bufs: make(map[string]*[]byte, 66), // Preallocate for ~66 Bible books
	}
}

func (x *xmlBookBytes) BookBytes(book_id string) *[]byte {
	p, ok := x.bufs[book_id]
	if !ok {
		s := make([]byte, 0, 4096) // Initial capacity for typical book XML
		p = &s
		x.bufs[book_id] = p
	}
	return p
}

func (x *xmlBookBytes) BookByteWriter(book_id string) io.ByteWriter {
	return &sliceByteWriter{p: x.BookBytes(book_id)}
}

type sliceByteWriter struct {
	p *[]byte // pointer to the slice for in-place append
}

func (w *sliceByteWriter) WriteByte(c byte) error {
	*w.p = append(*w.p, c)
	return nil
}

// ——————————————————————————————————————————————————————————————————————
// IXMLChunk implementation (shared underlying array, zero-copy)
// ——————————————————————————————————————————————————————————————————————
type xmlChunk struct {
	xml   []byte // shared reference to the full book XML
	start uint   // inclusive start index
	size  uint8  // length of the fragment (max 255)
}

func (c *xmlChunk) Bytes() iter.Seq[byte] {
	return func(yield func(byte) bool) {
		end := c.Start() + uint(c.ChunkSize())
		for i := c.Start(); i < end; i++ {
			if !yield(c.xml[i]) {
				return
			}
		}
	}
}

func (c *xmlChunk) ChunkSize() uint8 { return c.size }
func (c *xmlChunk) Start() uint      { return c.start }
func (c *xmlChunk) End() uint        { return c.Start() + uint(c.ChunkSize()) }

func (c *xmlChunk) String() string {
	const maxDisplay = 120
	b := c.xml[c.start:][:c.size]
	s := string(b)
	if len(s) > maxDisplay {
		s = s[:maxDisplay-3] + "..."
	}
	return fmt.Sprintf("%q (pos %d-%d, size %d)", s, c.start, c.start+uint(c.size)-1, c.size)
}

func newXMLChunk(xml []byte, start uint, size uint8) IXMLChunk {
	if start >= uint(len(xml)) {
		start = uint(len(xml))
	}
	if int(start)+int(size) > len(xml) {
		size = uint8(len(xml) - int(start))
	}
	if size == 0 {
		size = 1 // never return zero-size chunk
	}
	return &xmlChunk{xml: xml, start: start, size: size}
}
