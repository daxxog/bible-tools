package main

import (
	"iter"
	"fmt"
)

type book struct {
	id       string
	chapters []*chapter
}

func (b *book) ID() string { return b.id }
func (b *book) Chapters() iter.Seq[IChapter] {
	return func(yield func(IChapter) bool) {
		for _, c := range b.chapters {
			if !yield(c) {
				return
			}
		}
	}
}
func (b *book) Verses() iter.Seq[IVerse] {
	return func(yield func(IVerse) bool) {
		for _, c := range b.chapters {
			for _, v := range c.verses {
				if !yield(v) {
					return
				}
			}
		}
	}
}
func (b *book) Words() iter.Seq[IWord] {
	return func(yield func(IWord) bool) {
		for _, c := range b.chapters {
			for _, v := range c.verses {
				for _, w := range v.words {
					if !yield(w) {
						return
					}
				}
			}
		}
	}
}

type chapter struct {
	book   IBook
	number uint8
	verses []*verse
}

func (c *chapter) ID() string    { return fmt.Sprintf("%s.%d", c.book.ID(), c.number) }
func (c *chapter) Number() uint8 { return c.number }
func (c *chapter) Book() IBook   { return c.book }
func (c *chapter) Verses() iter.Seq[IVerse] {
	return func(yield func(IVerse) bool) {
		for _, v := range c.verses {
			if !yield(v) {
				return
			}
		}
	}
}
func (c *chapter) Words() iter.Seq[IWord] {
	return func(yield func(IWord) bool) {
		for _, v := range c.verses {
			for _, w := range v.words {
				if !yield(w) {
					return
				}
			}
		}
	}
}

type verse struct {
	book    IBook
	chapter IChapter
	number  uint8
	words   []*word
}

func (v *verse) ID() string        { return fmt.Sprintf("%s.%d", v.chapter.ID(), v.number) }
func (v *verse) Number() uint8     { return v.number }
func (v *verse) Book() IBook       { return v.book }
func (v *verse) Chapter() IChapter { return v.chapter }
func (v *verse) Words() iter.Seq[IWord] {
	return func(yield func(IWord) bool) {
		for _, w := range v.words {
			if !yield(w) {
				return
			}
		}
	}
}
