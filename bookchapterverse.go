package main

import (
	"fmt"
	"iter"
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

func leadingZero(number uint8) string {
	switch number {
	case 0:
		return "00"
	case 1:
		return "01"
	case 2:
		return "02"
	case 3:
		return "03"
	case 4:
		return "04"
	case 5:
		return "05"
	case 6:
		return "06"
	case 7:
		return "07"
	case 8:
		return "08"
	case 9:
		return "09"
	}

	return fmt.Sprintf("%d", number)
}

func (v *verse) ID() string        { return fmt.Sprintf("%s.%s", v.chapter.ID(), leadingZero(v.number)) }
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
