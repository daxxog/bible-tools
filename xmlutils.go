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

// BookBytes returns a pointer to the byte slice for the given book.
// It creates the slice if it does not already exist.
func (x *xmlBookBytes) BookBytes(book_id string) *[]byte {
	p, ok := x.bufs[book_id]
	if !ok {
		s := make([]byte, 0, 4096) // Initial capacity for typical book XML
		p = &s
		x.bufs[book_id] = p
	}
	return p
}

// BookBytesLen returns the current length of the byte buffer for the given book.
// It guarantees the buffer exists (creates it if necessary) before returning the length.
func (x *xmlBookBytes) BookBytesLen(book_id string) uint {
	return uint(len(*x.BookBytes(book_id)))
}

// BookByteWriter returns an io.ByteWriter that appends directly to the book's buffer.
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
	xml   *[]byte // pointer to the slice (shared across all chunks of a book)
	start uint    // inclusive start index
	size  uint8   // length of the fragment (max 255)
}

func (c *xmlChunk) Bytes() iter.Seq[byte] {
	return func(yield func(byte) bool) {
		end := c.start + uint(c.size)
		for i := c.start; i < end; i++ {
			if !yield((*c.xml)[i]) {
				return
			}
		}
	}
}

func (c *xmlChunk) ChunkSize() uint8 { return c.size }
func (c *xmlChunk) Start() uint      { return c.start }
func (c *xmlChunk) End() uint        { return c.start + uint(c.size) }

func (c *xmlChunk) String() string {
	const maxDisplay = 256
	b := (*c.xml)[c.start : c.start+uint(c.size)]
	s := string(b)
	if len(s) > maxDisplay {
		s = s[:maxDisplay-3] + "..."
	}
	return fmt.Sprintf("%q (pos %d-%d, size %d)", s, c.start, c.start+uint(c.size)-1, c.size)
}

// newXMLChunk creates a chunk from the shared book buffer.
func newXMLChunk(xml *[]byte, start uint, size uint8) IXMLChunk {
	currlen := uint(len(*xml))
	if start >= currlen {
		start = currlen
	}
	if start+uint(size) > currlen {
		size = uint8(currlen - start)
	}
	if size == 0 {
		size = 1 // never return zero-size chunk
	}
	return &xmlChunk{xml: xml, start: start, size: size}
}

// MergeChunks safely merges two chunks that are guaranteed to belong to the same book buffer.
func MergeChunks(a, b IXMLChunk) IXMLChunk {
	aa := a.(*xmlChunk)
	bb := b.(*xmlChunk)
	if aa.xml != bb.xml {
		panic("chunks from different XML sources")
	}
	m_start := min(aa.start, bb.start)
	m_end := max(aa.End(), bb.End())
	m_size := m_end - m_start
	if m_size > 255 {
		m_size = 255
		m_end = m_start + 255
	}
	return newXMLChunk(aa.xml, m_start, uint8(m_size))
}

func min(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

func max(a, b uint) uint {
	if a > b {
		return a
	}
	return b
}
