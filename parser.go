package main

import (
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"
	"unicode"

	"github.com/beevik/etree"
)

type devNullWriter struct { }
func (*devNullWriter) Write(p []byte) (n int, err error) { return len(p), nil }
func DevNullWriter() io.Writer {
	return &devNullWriter{}
}

// parseState defines the states for the XML parser state machine.
type parseState int

const (
	stateText parseState = iota
	stateTagOpen
	stateTagName
	stateAttrSpace
	stateAttrName
	stateAttrEq
	stateAttrValue
	stateSelfClose
)

type ETreeBookParserState struct {
	book            string // three character book id (e.g. "GEN", "JHN")
	builder         IBookBuilder // builder for the current book
	vopen           bool // inner verse parser (v tag open)
	wopen           bool // inner word parser (w tag open)
	vref            string // current verse reference (e.g. "JHN.21.4")
	sref            string // current strongs reference, reset to "" (nil IStrongsNumber) when wopen is set to false
	current_chapter uint8

	// Parser state
	parse_state     parseState
	is_closing      bool
	current_tag     string
	attrs           map[string]string
	current_attr_name string
	current_attr_value string
	current_buffer  strings.Builder // for text inside w
	pending_text    strings.Builder // for text outside w
}

type ETreeBookParser struct {
	debug_writer io.Writer
	state        *ETreeBookParserState
}

func NewEtreeParser() *ETreeBookParser {
	return NewEtreeParserWithDebug(DevNullWriter())
}

func NewEtreeParserWithDebug(debug_writer io.Writer) *ETreeBookParser {
	return &ETreeBookParser{debug_writer: debug_writer, state: &ETreeBookParserState{
		parse_state: stateText,
	}}
}

func (self *ETreeBookParser) SetBook(book string) {
	self.state.book = book
	self.state.builder = NewBookBuilder(book)
}

func (self *ETreeBookParser) WriteSettings() etree.WriteSettings {
	return etree.WriteSettings{
		CanonicalEndTags: false,
		CanonicalText:    false,
		CanonicalAttrVal: false,
		AttrSingleQuote:  false,
	}
}

func (self *ETreeBookParser) WriteByte(c byte) error {
	// Debug output
	var b [1]byte
	b[0] = c
	_, err := fmt.Fprintf(self.debug_writer, "WriteByte(%b, %q)\n", b, c)
	if err != nil {
		return err
	}
	return self.parseByte(c)
}

func (self *ETreeBookParser) parseByte(c byte) error {
	// Parse logic
	switch self.state.parse_state {
	case stateText:
		if c == '<' {
			// Process pending text outside w
			if self.state.vopen && !self.state.wopen {
				for f := range strings.FieldsSeq(self.state.pending_text.String()) {
					add_err := self.state.builder.AddWord(self.state.vref, "", f)
					if add_err != nil {
						return add_err
					}
				}
				self.state.pending_text.Reset()
			}
			self.state.parse_state = stateTagOpen
			self.state.is_closing = false
			self.state.current_tag = ""
			self.state.attrs = make(map[string]string)
			self.state.current_attr_name = ""
			self.state.current_attr_value = ""
		} else {
			if self.state.wopen {
				self.state.current_buffer.WriteByte(c)
			} else if self.state.vopen {
				self.state.pending_text.WriteByte(c)
			}
			// Ignore text outside verse
		}
	case stateTagOpen:
		if c == '/' {
			self.state.is_closing = true
		} else {
			self.state.current_tag = string(c)
			self.state.parse_state = stateTagName
		}
	case stateTagName:
		switch c {
		case ' ':
			if !self.state.is_closing {
				self.state.parse_state = stateAttrSpace
			} else {
				return fmt.Errorf("unexpected space in closing tag")
			}
		case '>':
			self.state.parse_state = stateText
			if self.state.is_closing {
				return self.handleCloseTag(self.state.current_tag)
			} else {
				return self.handleOpenTag(self.state.current_tag, self.state.attrs)
			}
		case '/':
			if self.state.is_closing {
				return fmt.Errorf("unexpected / in closing tag")
			}
			self.state.parse_state = stateSelfClose
		default:
			self.state.current_tag += string(c)
		}
	case stateAttrSpace:
		switch c {
		case ' ':
			// Skip spaces
		case '>':
			self.state.parse_state = stateText
			return self.handleOpenTag(self.state.current_tag, self.state.attrs)
		case '/':
			self.state.parse_state = stateSelfClose
		default:
			self.state.current_attr_name = string(c)
			self.state.parse_state = stateAttrName
		}
	case stateAttrName:
		if c == '=' {
			self.state.parse_state = stateAttrEq
		} else {
			self.state.current_attr_name += string(c)
		}
	case stateAttrEq:
		if c == '"' {
			self.state.current_attr_value = ""
			self.state.parse_state = stateAttrValue
		} else {
			return fmt.Errorf("unexpected character after =")
		}
	case stateAttrValue:
		if c == '"' {
			self.state.attrs[self.state.current_attr_name] = self.state.current_attr_value
			self.state.parse_state = stateAttrSpace
		} else {
			self.state.current_attr_value += string(c)
		}
	case stateSelfClose:
		if c == '>' {
			self.state.parse_state = stateText
			return self.handleOpenTag(self.state.current_tag, self.state.attrs)
		} else {
			return fmt.Errorf("expected > after / in self-closing tag")
		}
	}
	return nil
}

func (self *ETreeBookParser) WriteString(s string) (int, error) {
	// Debug
	n, err := fmt.Fprintf(self.debug_writer, "WriteString(%q)\n", s)
	if err != nil {
		return n, err
	}
	// Parse
	for _, c := range []byte(s) {
		if err := self.parseByte(c); err != nil {
			return 0, err
		}
	}
	return len(s), nil
}

func (self *ETreeBookParser) Write(b []byte) (int, error) {
	// Debug
	n, err := fmt.Fprintf(self.debug_writer, "Write(%q)\n", b)
	if err != nil {
		return n, err
	}
	// Parse
	for _, c := range b {
		if err := self.parseByte(c); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (self *ETreeBookParser) Flush() error {
	if self.state.vopen && !self.state.wopen {
		for f := range strings.FieldsSeq(self.state.pending_text.String()) {
			add_err := self.state.builder.AddWord(self.state.vref, "", f)
			if add_err != nil {
				return add_err
			}
		}
		self.state.pending_text.Reset()
	}
	return nil
}

func (self *ETreeBookParser) handleOpenTag(tag string, attrs map[string]string) error {
	switch tag {
	case "c":
		idstr := attrs["id"]
		n, err := strconv.ParseUint(idstr, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid chapter id: %s", idstr)
		}
		self.state.current_chapter = uint8(n)
	case "v":
		idstr := attrs["id"]
		n, err := strconv.ParseUint(idstr, 10, 8)
		if err != nil {
			return fmt.Errorf("invalid verse id: %s", idstr)
		}
		self.state.vref = fmt.Sprintf("%s.%d.%d", self.state.book, self.state.current_chapter, n)
		self.state.vopen = true
	case "w":
		self.state.sref = attrs["s"]
		self.state.wopen = true
		self.state.current_buffer.Reset()
	case "ve":
		self.state.vopen = false
	}
	return nil
}

func (self *ETreeBookParser) handleCloseTag(tag string) error {
	switch tag {
	case "c":
		// No action needed
	case "v":
		self.state.vopen = false
	case "w":
		fulltext := strings.TrimSpace(self.state.current_buffer.String())
		if fulltext != "" {
			add_err := self.state.builder.AddWord(self.state.vref, self.state.sref, fulltext)
			if add_err != nil {
				return add_err
			}
		}
		self.state.wopen = false
		self.state.sref = ""
	}
	return nil
}

// strongsNumber implements IStrongsNumber.
type strongsNumber struct {
	strongs_type string
	value        uint16
}

func (s *strongsNumber) Type() string   { return s.strongs_type }
func (s *strongsNumber) Value() uint16  { return s.value }
func (s *strongsNumber) String() string {
	if s.value == 0 {
		return ""
	}
	return fmt.Sprintf("%s%d", s.strongs_type, s.value)
}

// parseStrongs parses a Strong's reference.
func parseStrongs(raw string) IStrongsNumber {
	if raw == "" {
		return &strongsNumber{}
	}
	if len(raw) < 2 {
		return &strongsNumber{}
	}
	t := string(raw[0])
	if t != "H" && t != "G" {
		return &strongsNumber{}
	}
	v, err := strconv.ParseUint(raw[1:], 10, 16)
	if err != nil {
		return &strongsNumber{}
	}
	return &strongsNumber{strongs_type: t, value: uint16(v)}
}

// bookBuilder implements IBookBuilder.
type bookBuilder struct {
	book            *book
	last_chapter    uint8
	last_verse      uint8
	current_chapter *chapter
	current_verse   *verse
}

func NewBookBuilder(id string) IBookBuilder {
	return &bookBuilder{
		book: &book{id: id, chapters: []*chapter{}},
	}
}

func (b *bookBuilder) ID() string {
	return b.book.id
}

func (b *bookBuilder) AddWord(vref string, sref string, word_fulltext string) error {
	parts := strings.Split(vref, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid vref: %s", vref)
	}
	book_id := parts[0]
	c_str := parts[1]
	v_str := parts[2]
	if book_id != b.book.id {
		return fmt.Errorf("word for wrong book: %s != %s", book_id, b.book.id)
	}
	c_num, err := strconv.ParseUint(c_str, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid chapter in vref: %s", vref)
	}
	v_num, err := strconv.ParseUint(v_str, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid verse in vref: %s", vref)
	}
	strongs := parseStrongs(sref)
	if sref != "" && strongs.Type() == "" {
		return fmt.Errorf("invalid strongs reference: %s", sref)
	}
	cu8 := uint8(c_num)
	vu8 := uint8(v_num)
	if cu8 != b.last_chapter {
		if cu8 < b.last_chapter {
			return fmt.Errorf("chapters out of order: %d < %d [book.id=%s]", cu8, b.last_chapter, b.book.ID())
		}
		new_ch := &chapter{book: b.book, number: cu8}
		b.book.chapters = append(b.book.chapters, new_ch)
		b.current_chapter = new_ch
		b.last_chapter = cu8
		b.last_verse = 0
		b.current_verse = nil
	}
	if vu8 != b.last_verse {
		if vu8 < b.last_verse {
			return fmt.Errorf("verses out of order: %d < %d [book.id=%s]", vu8, b.last_verse, b.book.ID())
		}
		new_vs := &verse{book: b.book, chapter: b.current_chapter, number: vu8}
		b.current_chapter.verses = append(b.current_chapter.verses, new_vs)
		b.current_verse = new_vs
		b.last_verse = vu8
	}
	new_word := &word{
		book:      b.book,
		chapter:   b.current_chapter,
		verse:     b.current_verse,
		strongs:   strongs,
		full_text: word_fulltext,
	}
	b.current_verse.words = append(b.current_verse.words, new_word)
	return nil
}

func (b *bookBuilder) Build() IBook {
	return b.book
}

// Implementations for book, chapter, verse, word (as in previous response)
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
	book    IBook
	number  uint8
	verses  []*verse
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
	book     IBook
	chapter  IChapter
	number   uint8
	words    []*word
}

func (v *verse) ID() string     { return fmt.Sprintf("%s.%d", v.chapter.ID(), v.number) }
func (v *verse) Number() uint8  { return v.number }
func (v *verse) Book() IBook    { return v.book }
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

type word struct {
	book      IBook
	chapter   IChapter
	verse     IVerse
	strongs   IStrongsNumber
	full_text string
}

func (w *word) Book() IBook                  { return w.book }
func (w *word) Chapter() IChapter            { return w.chapter }
func (w *word) Verse() IVerse                { return w.verse }
func (w *word) StrongsNumber() IStrongsNumber { return w.strongs }
func (w *word) FullText() string             { return w.full_text }
func (w *word) Text() string {
	var builder strings.Builder
	for _, r := range strings.ToLower(w.full_text) {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
