package main

import (
	"fmt"
	"io"
	"iter"

	"github.com/beevik/etree"
)

type IBook interface {
	ID() string // three character book id (e.g. "GEN", "JHN")
	Chapters() iter.Seq[IChapter] // sequence of chapters contained in this book
	Verses() iter.Seq[IVerse] // sequence of verses contained in this book
	Words() iter.Seq[IWord] // sequence of words contained in this book
}

type IChapter interface {
	ID() string // e.g. JHN.21
	Number() uint8 // Chapter number
	Book() IBook // book this chapter is apart of
	Verses() iter.Seq[IVerse] // sequence of verses contained in this chapter
	Words() iter.Seq[IWord] // sequence of words contained in this chapter
}

type IVerse interface {
	ID() string // e.g. GEN.1.1
	Number() uint8 // Verse number
	Book() IBook // book this verse is apart of
	Chapter() IChapter // chapter this verse is apart of
	Words() iter.Seq[IWord] // sequence of words contained in this verse
}

type IWord interface {
	Book() IBook // book this word is apart of
	Chapter() IChapter // chapter this word is apart of
	Verse() IVerse // verse this word is apart of
	StrongsNumber() IStrongsNumber // concordance reference
	Text() string // Lowercase text stripped of whitespace and punctuation
	FullText() string // Text (with punctuation / capitalization), whitespace preserved
}

type IStrongsNumber interface {
	Type() string // "H" for Hebrew, "G" for Greek, "" (undefined / not found)
	Value() uint16 // number or 0 (undefined / not found)
	fmt.Stringer // raw string representation (e.g. "H6212", "G1456") [empty string "" for nil]
}

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
