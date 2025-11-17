package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/beevik/etree"
)

type devNullWriter struct{}

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
	book            string       // three character book id (e.g. "GEN", "JHN")
	builder         IBookBuilder // builder for the current book
	vopen           bool         // inner verse parser (v tag open)
	wopen           bool         // inner word parser (w tag open)
	vref            string       // current verse reference (e.g. "JHN.21.4")
	sref            string       // current strongs reference, reset to "" when wopen = false
	current_chapter uint8
	note_depth      int // to skip footnotes

	// Parser state
	parse_state        parseState
	is_closing         bool
	current_tag        string
	attrs              map[string]string
	current_attr_name  string
	current_attr_value string
	current_buffer     strings.Builder // for text inside w
	pending_text       strings.Builder // for text outside w
	pending_start      uint            // start pos for pending_text or w tag
	tag_start_pos      uint            // start pos of current tag (<)
}

type ETreeBookParser struct {
	debug_writer io.Writer
	state        *ETreeBookParserState
	book_bytes   IXMLBookBytes // singleton manager for per-book XML byte arrays
}

// ——————————————————————————————————————————————————————————————————————
// ETreeBookParser factory
// ——————————————————————————————————————————————————————————————————————
func NewEtreeParser() *ETreeBookParser {
	return NewEtreeParserWithDebug(DevNullWriter())
}

func NewEtreeParserWithDebug(debug_writer io.Writer) *ETreeBookParser {
	return &ETreeBookParser{
		debug_writer: debug_writer,
		state: &ETreeBookParserState{
			parse_state:   stateText,
			pending_start: 0,
			tag_start_pos: 0,
		},
		book_bytes: newXMLBookBytes(),
	}
}

func (self *ETreeBookParser) SetBook(book string) {
	self.state.book = book
	self.state.builder = NewBookBuilder(book)
	self.state.pending_start = self.book_bytes.BookBytesLen(book)
	self.state.tag_start_pos = 0

	// Reset parser state
	self.state.current_chapter = 0
	self.state.vopen = false
	self.state.wopen = false
	self.state.vref = ""
	self.state.sref = ""
	self.state.note_depth = 0
	self.state.is_closing = false
	self.state.current_tag = ""
	self.state.attrs = nil
	self.state.current_attr_name = ""
	self.state.current_attr_value = ""
	self.state.current_buffer.Reset()
	self.state.pending_text.Reset()
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
	if err := self.book_bytes.BookByteWriter(self.state.book).WriteByte(c); err != nil {
		return err
	}
	pos := self.book_bytes.BookBytesLen(self.state.book) - 1
	_, err := fmt.Fprintf(self.debug_writer, "WriteByte(%b, %q) pos=%d\n", c, c, pos)
	if err != nil {
		return err
	}
	return self.parseByte(c, pos)
}

func (self *ETreeBookParser) WriteString(s string) (int, error) {
	_, err := fmt.Fprintf(self.debug_writer, "WriteString(%q)\n", s)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(s); i++ {
		if err := self.WriteByte(s[i]); err != nil {
			return i, err
		}
	}
	return len(s), nil
}

func (self *ETreeBookParser) Write(b []byte) (int, error) {
	_, err := fmt.Fprintf(self.debug_writer, "Write(%q)\n", b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < len(b); i++ {
		if err := self.WriteByte(b[i]); err != nil {
			return i, err
		}
	}
	return len(b), nil
}

func (self *ETreeBookParser) Flush() (IBook, error) {
	if self.state.builder == nil {
		return nil, errors.New("nil book builder (was .Flush called twice or without SetBook?)")
	}
	if self.state.vopen && !self.state.wopen && self.state.note_depth == 0 {
		end_pos := self.book_bytes.BookBytesLen(self.state.book)
		if err := self.flushPendingText(end_pos); err != nil {
			return nil, err
		}
	}
	book := self.state.builder.Build()
	self.state.builder = nil
	return book, nil
}

// ——————————————————————————————————————————————————————————————————————
// Core parsing logic
// ——————————————————————————————————————————————————————————————————————
func (self *ETreeBookParser) parseByte(c byte, pos uint) error {
	if self.state.builder == nil {
		return errors.New("nil book builder, must call SetBook before parsing!")
	}

	switch self.state.parse_state {
	case stateText:
		if c == '<' {
			if self.state.vopen && !self.state.wopen && self.state.note_depth == 0 {
				if err := self.flushPendingText(pos); err != nil {
					return err
				}
			}
			self.state.tag_start_pos = pos
			self.state.parse_state = stateTagOpen
			self.state.is_closing = false
			self.state.current_tag = ""
			self.state.attrs = make(map[string]string)
			self.state.current_attr_name = ""
			self.state.current_attr_value = ""
		} else {
			if self.state.wopen && self.state.note_depth == 0 {
				self.state.current_buffer.WriteByte(c)
			} else if self.state.vopen && self.state.note_depth == 0 {
				if self.state.pending_text.Len() == 0 {
					self.state.pending_start = pos
				}
				self.state.pending_text.WriteByte(c)
			}
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
				return self.handleCloseTag(self.state.current_tag, pos)
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
			// skip
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
		if self.state.note_depth == 0 {
			self.state.sref = attrs["s"]
			self.state.wopen = true
			self.state.current_buffer.Reset()
			self.state.pending_start = self.state.tag_start_pos
		}
	case "ve":
		self.state.vopen = false
	case "f", "x":
		self.state.note_depth++
	}
	return nil
}

func (self *ETreeBookParser) handleCloseTag(tag string, pos uint) error {
	switch tag {
	case "c":
		// No action
	case "v":
		self.state.vopen = false
	case "w":
		if self.state.note_depth == 0 {
			fulltext := strings.TrimSpace(self.state.current_buffer.String())
			if fulltext != "" {
				const context uint = 20
				start := self.state.pending_start
				if start > context {
					start -= context
				}
				chunk_size := uint8(pos - start + 1 + context)
				chunk := newXMLChunk(self.book_bytes.BookBytes(self.state.book), start, chunk_size)
				add_err := self.state.builder.AddWord(self.state.vref, self.state.sref, fulltext, chunk)
				if add_err != nil {
					return add_err
				}
			}
			self.state.wopen = false
			self.state.sref = ""
		}
	case "f", "x":
		self.state.note_depth--
	}
	return nil
}

// ——————————————————————————————————————————————————————————————————————
// Helper: contraction merging
// ——————————————————————————————————————————————————————————————————————
func isAllPunct(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsPunct(r) {
			return false
		}
	}
	return true
}

func isContractionSuffix(suffix string) bool {
	switch strings.ToLower(suffix) {
	case "s", "t", "re", "ll", "ve", "d", "m", "em", "clock":
		return true
	}
	return false
}

// Add this struct near flushPendingText
type wordInfo struct {
	text  string
	start uint
	end   uint
}

func (self *ETreeBookParser) flushPendingText(end_pos uint) error {
	if !self.state.vopen || self.state.wopen || self.state.note_depth != 0 {
		return nil
	}
	s := self.state.pending_text.String()
	if s == "" {
		return nil
	}
	var wordInfos []wordInfo
	offset := uint(0)
	for offset < uint(len(s)) {
		r, size := utf8.DecodeRuneInString(s[int(offset):])
		if unicode.IsSpace(r) {
			offset += uint(size)
			continue
		}
		wstart := self.state.pending_start + offset
		var builder strings.Builder
		for offset < uint(len(s)) {
			r, size := utf8.DecodeRuneInString(s[int(offset):])
			if unicode.IsSpace(r) {
				break
			}
			builder.WriteRune(r)
			offset += uint(size)
		}
		wend := self.state.pending_start + offset
		wordInfos = append(wordInfos, wordInfo{
			text:  builder.String(),
			start: wstart,
			end:   wend,
		})
	}
	merged := mergeContractionsWithPos(wordInfos)
	for _, wi := range merged {
		context := uint(60)
		cstart := wi.start
		if cstart > context {
			cstart -= context
		}
		cend := wi.end + context
		if cend > self.book_bytes.BookBytesLen(self.state.book) {
			cend = self.book_bytes.BookBytesLen(self.state.book)
		}
		csize := cend - cstart
		if csize > 255 {
			wordLen := wi.end - wi.start
			extra := uint(255) - wordLen
			before := extra / 2
			after := extra - before
			cstart = wi.start
			if cstart > before {
				cstart -= before
			} else {
				cstart = 0
			}
			cend = wi.end + after
			if cend > self.book_bytes.BookBytesLen(self.state.book) {
				cend = self.book_bytes.BookBytesLen(self.state.book)
				cstart = cend - 255
			}
			csize = cend - cstart
		}
		chunk := newXMLChunk(self.book_bytes.BookBytes(self.state.book), cstart, uint8(csize))
		add_err := self.state.builder.AddWord(self.state.vref, "", wi.text, chunk)
		if add_err != nil {
			return add_err
		}
	}
	self.state.pending_text.Reset()
	self.state.pending_start = end_pos
	return nil
}

// Updated helper: contraction merging with positions
func mergeContractionsWithPos(infos []wordInfo) []wordInfo {
	var merged []wordInfo
	for i := 0; i < len(infos); i++ {
		f := infos[i]
		if len(merged) > 0 {
			prevIdx := len(merged) - 1
			prev := merged[prevIdx]
			if strings.HasSuffix(prev.text, "’") || strings.HasSuffix(prev.text, "'") {
				if isContractionSuffix(f.text) {
					merged[prevIdx].text += f.text
					merged[prevIdx].end = f.end
					continue
				}
			}
			r, _ := utf8.DecodeRuneInString(f.text)
			if r == '’' || r == '\'' {
				suffix := f.text[utf8.RuneLen(r):]
				if isContractionSuffix(suffix) {
					merged[prevIdx].text += f.text
					merged[prevIdx].end = f.end
					continue
				}
			}
		}
		merged = append(merged, f)
	}
	return merged
}

// ——————————————————————————————————————————————————————————————————————
// bookBuilder implements IBookBuilder
// ——————————————————————————————————————————————————————————————————————
type bookBuilder struct {
	book            *book
	last_chapter    uint8
	last_verse      uint8
	current_chapter *chapter
	current_verse   *verse
	last_word       *word
}

func NewBookBuilder(id string) IBookBuilder {
	return &bookBuilder{
		book: &book{id: id, chapters: []*chapter{}},
	}
}

func (b *bookBuilder) ID() string {
	return b.book.id
}

// Updated AddWord to merge chunks when appending text
func (b *bookBuilder) AddWord(vref string, sref string, word_fulltext string, chunk IXMLChunk) error {
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
		b.last_word = nil
	}
	if vu8 != b.last_verse {
		if vu8 < b.last_verse {
			return fmt.Errorf("verses out of order: %d < %d [book.id=%s]", vu8, b.last_verse, b.book.ID())
		}
		new_vs := &verse{book: b.book, chapter: b.current_chapter, number: vu8}
		b.current_chapter.verses = append(b.current_chapter.verses, new_vs)
		b.current_verse = new_vs
		b.last_verse = vu8
		b.last_word = nil
	}

	// ———————————————————————————
	// 1. Merge ’s, ’t, etc. with previous word
	// ———————————————————————————
	if b.last_word != nil {
		last := b.last_word.full_text
		if len(last) > 0 && unicode.IsLetter(rune(last[len(last)-1])) {
			if word_fulltext == "’s" || word_fulltext == "'s" {
				b.last_word.full_text += word_fulltext
				b.last_word.xml_chunk = MergeChunks(b.last_word.xml_chunk, chunk)
				return nil
			}
			if strings.HasSuffix(last, "n") && (word_fulltext == "’t" || word_fulltext == "'t") {
				b.last_word.full_text += word_fulltext
				if strongs.Type() != "" && b.last_word.strongs.Type() == "" {
					b.last_word.strongs = strongs
				}
				b.last_word.xml_chunk = MergeChunks(b.last_word.xml_chunk, chunk)
				return nil
			}
		}
		// Existing contraction logic (e.g. ’re, ’ll)
		if strings.HasSuffix(last, "’") || strings.HasSuffix(last, "'") {
			if isContractionSuffix(word_fulltext) {
				b.last_word.full_text += word_fulltext
				if strongs.Type() != "" && b.last_word.strongs.Type() == "" {
					b.last_word.strongs = strongs
				}
				b.last_word.xml_chunk = MergeChunks(b.last_word.xml_chunk, chunk)
				return nil
			}
		}
		if strings.HasPrefix(word_fulltext, "’") || strings.HasPrefix(word_fulltext, "'") {
			if isContractionSuffix(word_fulltext[1:]) {
				b.last_word.full_text += word_fulltext
				if strongs.Type() != "" && b.last_word.strongs.Type() == "" {
					b.last_word.strongs = strongs
				}
				b.last_word.xml_chunk = MergeChunks(b.last_word.xml_chunk, chunk)
				return nil
			}
		}
	}

	// ———————————————————————————
	// 2. Punctuation
	// ———————————————————————————
	if isAllPunct(word_fulltext) {
		if b.last_word != nil {
			b.last_word.full_text += word_fulltext
			b.last_word.xml_chunk = MergeChunks(b.last_word.xml_chunk, chunk)
		}
		return nil
	}

	// ———————————————————————————
	// 3. New word
	// ———————————————————————————
	new_word := &word{
		book:      b.book,
		chapter:   b.current_chapter,
		verse:     b.current_verse,
		strongs:   strongs,
		full_text: word_fulltext,
		xml_chunk: chunk,
	}
	b.current_verse.words = append(b.current_verse.words, new_word)
	b.last_word = new_word
	return nil
}

func (b *bookBuilder) Build() IBook {
	return b.book
}

// ——————————————————————————————————————————————————————————————————————
// word implements IWord
// ——————————————————————————————————————————————————————————————————————
type word struct {
	book      IBook
	chapter   IChapter
	verse     IVerse
	strongs   IStrongsNumber
	full_text string
	xml_chunk IXMLChunk
}

func (w *word) Book() IBook                   { return w.book }
func (w *word) Chapter() IChapter             { return w.chapter }
func (w *word) Verse() IVerse                 { return w.verse }
func (w *word) StrongsNumber() IStrongsNumber { return w.strongs }
func (w *word) FullText() string              { return w.full_text }
func (w *word) Text() string {
	var builder strings.Builder
	for _, r := range strings.ToLower(w.full_text) {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
func (w *word) XMLChunk() IXMLChunk { return w.xml_chunk }
