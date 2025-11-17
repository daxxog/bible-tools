package main

import (
	"fmt"
	"io"
	"iter"
	"regexp"
)

type IBookIndex interface {
	Book(book_id string) (IBook, error)                          // lookup book by book_id
	FirstStrongs(strongs_number IStrongsNumber) IWord            // nil if not found
	LookupStrongs(strongs_number IStrongsNumber) iter.Seq[IWord] // lookup words by Strong's number:q
	LastStrongs(strongs_number IStrongsNumber) IWord             // nil if not found
	SearchEnglish(regexp *regexp.Regexp) iter.Seq[IWord]         // lookup words (matching on IWord.Text) using regular expression
	Books() iter.Seq[IBook]                                      // sequence of books contained in this index
	Chapters() iter.Seq[IChapter]                                // sequence of chapters contained in this index
	Verses() iter.Seq[IVerse]                                    // sequence of verses contained in this index
	Words() iter.Seq[IWord]                                      // sequence of words contained in this index
}

type IBookIndexBuilder interface {
	AddBook(IBook) error        // error if .AddBook called after .Build
	Build() (IBookIndex, error) // error if called twice
}

type IXMLBookBytes interface {
	BookBytes(book_id string) *[]byte            // return a reference to the existing byte array or initialize a new one (Singleton pattern)
	BookBytesLen(book_id string) uint            // current length of underlying byte array
	BookByteWriter(book_id string) io.ByteWriter // return a writer which can safely append to the underlying byte array (creating one if needed) for a specific book_id
}

type IBook interface {
	ID() string                   // three character book id (e.g. "GEN", "JHN")
	Chapters() iter.Seq[IChapter] // sequence of chapters contained in this book
	Verses() iter.Seq[IVerse]     // sequence of verses contained in this book
	Words() iter.Seq[IWord]       // sequence of words contained in this book
}

type IChapter interface {
	ID() string               // e.g. JHN.21
	Number() uint8            // Chapter number
	Book() IBook              // book this chapter is apart of
	Verses() iter.Seq[IVerse] // sequence of verses contained in this chapter
	Words() iter.Seq[IWord]   // sequence of words contained in this chapter
}

type IVerse interface {
	ID() string             // e.g. GEN.1.1
	Number() uint8          // Verse number
	Book() IBook            // book this verse is apart of
	Chapter() IChapter      // chapter this verse is apart of
	Words() iter.Seq[IWord] // sequence of words contained in this verse
}

type IWord interface {
	Book() IBook                   // book this word is apart of
	Chapter() IChapter             // chapter this word is apart of
	Verse() IVerse                 // verse this word is apart of
	StrongsNumber() IStrongsNumber // concordance reference
	Text() string                  // Lowercase text stripped of whitespace and punctuation
	FullText() string              // Text (with punctuation / capitalization), whitespace preserved
	XMLChunk() IXMLChunk           // chunk of XML this word was parsed from
}

type IStrongsNumber interface {
	Type() string  // "H" for Hebrew, "G" for Greek, "" (undefined / not found)
	Value() uint16 // number or 0 (undefined / not found)
	fmt.Stringer   // raw string representation (e.g. "H6212", "G1456") [empty string "" for nil]
}

type IXMLChunk interface {
	Bytes() iter.Seq[byte] // sequence of bytes representing the chunk of XML from the larger byte array
	ChunkSize() uint8      // size of the XML chunk—the goal when chunking is to include the entire word + enough before and after to reason about when debugging
	Start() uint           // starting position of this chunk in the complete XML byte array
	End() uint             // Start + ChunkSize
	fmt.Stringer           // string representation of this chunk of XML
}

type IBookBuilder interface {
	ID() string                                                                        // three character book id (e.g. "GEN", "JHN")
	AddWord(vref string, sref string, word_fulltext string, xml_chunk IXMLChunk) error // returns an error if attempting to add a word outside of the book, or if an invalid strongs reference is passed
	Build() IBook                                                                      // finalize the structure
}
