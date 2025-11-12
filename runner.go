package main

import (
	"fmt"
	"io"

	"github.com/beevik/etree"
)

func RunUSFX(in io.Reader, usfx IUSFX, out io.Writer) error {
	el, err := usfx.USFXRootElement()
	if err != nil {
		return fmt.Errorf("Could not find <usfx> Root Element: %w", err)
	}
	path := etree.MustCompilePath("[name()='book']")
	elements := el.FindElementsPathSeq(path)
	o := NewEtreeParser()
	ws := o.WriteSettings()
	for book := range elements {
		book_id := book.SelectAttrValue("id", "❌")
		if book_id == "❌" {
			return fmt.Errorf("<book id=NOT FOUND")
		}
		// if book_id == "GEN" { // testing just with one book for now
		o.SetBook(book_id)
		for _, token := range book.Child {
			token.WriteTo(o, &ws)
		}
		if book, err := o.Flush(); err == nil {
			fmt.Fprintln(out, book.ID())
			i := 0
			// dnw := DevNullWriter()
			xml_debug := false
			for word := range book.Words() {
				i++
				if xml_debug {
					fmt.Fprintf(out, "%d['%s'][\"%s\"]{%s}\t\t(%s)\n", i, word.Verse().ID(), word.FullText(), word.StrongsNumber(), word.XMLChunk())
				} else {
					fmt.Fprintf(out, "%d['%s'][\"%s\"]{%s}\n", i, word.Verse().ID(), word.FullText(), word.StrongsNumber())
				}
			}
		} else {
			return err
		}
		// }
	}

	return nil
}
