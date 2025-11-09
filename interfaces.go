package main

import (
	"fmt"
	"iter"
)

type IBook interface {
	ID() string                   // three character book id (e.g. "GEN", "JHN")
	Chapters() iter.Seq[IChapter] // sequence of chapters contained in this book
	Verses() iter.Seq[IVerse]     // sequence of verses contained in this book
	Words() iter.Seq[IWord]       // sequence of words contained in this book
}

type IChapter interface {
	ID() string                 // e.g. JHN.21
	Number() uint8              // Chapter number
	Book() IBook                // book this chapter is apart of
	Verses() iter.Seq[IVerse]   // sequence of verses contained in this chapter
	Words() iter.Seq[IWord]     // sequence of words contained in this chapter
}

type IVerse interface {
	ID() string               // e.g. GEN.1.1
	Number() uint8            // Verse number
	Book() IBook              // book this verse is apart of
	Chapter() IChapter        // chapter this verse is apart of
	Words() iter.Seq[IWord]   // sequence of words contained in this verse
}

type IWord interface {
	Book() IBook                  // book this word is apart of
	Chapter() IChapter            // chapter this word is apart of
	Verse() IVerse                // verse this word is apart of
	StrongsNumber() IStrongsNumber // concordance reference
	Text() string                 // Lowercase text stripped of whitespace and punctuation
	FullText() string             // Text (with punctuation / capitalization), whitespace preserved
}

type IStrongsNumber interface {
	Type() string // "H" for Hebrew, "G" for Greek, "" (undefined / not found)
	Value() uint16 // number or 0 (undefined / not found)
	fmt.Stringer // raw string representation (e.g. "H6212", "G1456") [empty string "" for nil]
}

type IBookBuilder interface {
	ID() string // three character book id (e.g. "GEN", "JHN")
	AddWord(vref string, sref string, word_fulltext string) error // returns an error if attempting to add a word outside of the book, or if an invalid strongs reference is passed
	Build() IBook // finalize the structure
}

