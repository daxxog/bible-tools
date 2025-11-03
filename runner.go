package main

import (
	"fmt"
	"io"

	"github.com/beevik/etree"
)

type ETreeBookParser struct {
	debug_writer io.Writer
}

func NewEtreeParser(debug_writer io.Writer) *ETreeBookParser {
	return &ETreeBookParser{debug_writer: debug_writer}
}

func (self *ETreeBookParser) WriteSettings() etree.WriteSettings {
	return etree.WriteSettings{
		CanonicalEndTags: false,
		CanonicalText: false,
		CanonicalAttrVal: false,
		AttrSingleQuote: false,
	}
}

func (self *ETreeBookParser) WriteByte(c byte) error {
	var b [1]byte
	b[0] = c
	_, err := fmt.Fprintf(self.debug_writer, "WriteByte(%b, %q)\n", b, c)
	return err
}

func (self *ETreeBookParser) WriteString(s string) (int, error) {
	return fmt.Fprintf(self.debug_writer, "WriteString(%q)\n", s)
}

func (self *ETreeBookParser) Write(b []byte) (int, error) {
	return fmt.Fprintf(self.debug_writer, "Write(%q)\n", b)
}

func RunUSFX(in io.Reader, usfx IUSFX, out io.Writer) error {
	el, err := usfx.USFXRootElement()
	if err != nil {
		return fmt.Errorf("Could not find <usfx> Root Element: %w", err)
	}

	path := etree.MustCompilePath("[name()='book']")
	elements := el.FindElementsPathSeq(path)
	o := NewEtreeParser(out)
	ws := o.WriteSettings()

	for book := range elements {
		book_id := book.SelectAttrValue("id", "❌")
		if book_id == "❌" {
			return fmt.Errorf("<book id=NOT FOUND")
		}

		for _, token := range book.Child {
			token.WriteTo(o, &ws)
		}
	}

	return nil
}
