package main

import (
	"errors"
	"fmt"
	"iter"
	"regexp"
)

// strongsKey is a hashable key for Strong's numbers.
type strongsKey struct {
	typ   string
	value uint16
}

// bookIndexBuilder implements IBookIndexBuilder.
type bookIndexBuilder struct {
	built    bool
	books    map[string]IBook
	strongs  map[strongsKey][]IWord
	allWords []IWord
}

func NewBookIndexBuilder() IBookIndexBuilder {
	return &bookIndexBuilder{
		books:   make(map[string]IBook),
		strongs: make(map[strongsKey][]IWord),
	}
}

func (b *bookIndexBuilder) AddBook(book IBook) error {
	if b.built {
		return errors.New("cannot add book after index has been built")
	}
	id := book.ID()
	if _, exists := b.books[id]; exists {
		return fmt.Errorf("duplicate book ID: %s", id)
	}
	b.books[id] = book

	for word := range book.Words() {
		b.allWords = append(b.allWords, word)
		sn := word.StrongsNumber()
		if sn.Type() != "" {
			key := strongsKey{typ: sn.Type(), value: sn.Value()}
			b.strongs[key] = append(b.strongs[key], word)
		}
	}
	return nil
}

func (b *bookIndexBuilder) Build() (IBookIndex, error) {
	if b.built {
		return nil, errors.New("build called multiple times")
	}
	b.built = true
	return &bookIndex{
		books:    b.books,
		strongs:  b.strongs,
		allWords: b.allWords,
	}, nil
}

// bookIndex implements IBookIndex.
// It is read-only and safe for concurrent use.
type bookIndex struct {
	books    map[string]IBook
	strongs  map[strongsKey][]IWord
	allWords []IWord
}

func (i *bookIndex) Book(book_id string) (IBook, error) {
	book, exists := i.books[book_id]
	if !exists {
		return nil, fmt.Errorf("book not found: %s", book_id)
	}
	return book, nil
}

func (i *bookIndex) FirstStrongs(strongs_number IStrongsNumber) IWord {
	if strongs_number.Type() == "" {
		return nil
	}
	key := strongsKey{typ: strongs_number.Type(), value: strongs_number.Value()}
	words := i.strongs[key]
	if len(words) == 0 {
		return nil
	}
	return words[0]
}

func (i *bookIndex) LastStrongs(strongs_number IStrongsNumber) IWord {
	if strongs_number.Type() == "" {
		return nil
	}
	key := strongsKey{typ: strongs_number.Type(), value: strongs_number.Value()}
	words := i.strongs[key]
	if len(words) == 0 {
		return nil
	}
	return words[len(words)-1]
}

func (i *bookIndex) LookupStrongs(strongs_number IStrongsNumber) iter.Seq[IWord] {
	return func(yield func(IWord) bool) {
		if strongs_number.Type() == "" {
			return
		}
		key := strongsKey{typ: strongs_number.Type(), value: strongs_number.Value()}
		for _, word := range i.strongs[key] {
			if !yield(word) {
				return
			}
		}
	}
}

func (i *bookIndex) SearchEnglish(regexp *regexp.Regexp) iter.Seq[IWord] {
	return func(yield func(IWord) bool) {
		for _, word := range i.allWords {
			if regexp.MatchString(word.Text()) {
				if !yield(word) {
					return
				}
			}
		}
	}
}

func (i *bookIndex) Books() iter.Seq[IBook] {
	return func(yield func(IBook) bool) {
		for _, book := range i.books {
			if !yield(book) {
				return
			}
		}
	}
}

func (i *bookIndex) Chapters() iter.Seq[IChapter] {
	return func(yield func(IChapter) bool) {
		for _, book := range i.books {
			for chapter := range book.Chapters() {
				if !yield(chapter) {
					return
				}
			}
		}
	}
}

func (i *bookIndex) Verses() iter.Seq[IVerse] {
	return func(yield func(IVerse) bool) {
		for _, book := range i.books {
			for verse := range book.Verses() {
				if !yield(verse) {
					return
				}
			}
		}
	}
}

func (i *bookIndex) Words() iter.Seq[IWord] {
	return func(yield func(IWord) bool) {
		for _, word := range i.allWords {
			if !yield(word) {
				return
			}
		}
	}
}
