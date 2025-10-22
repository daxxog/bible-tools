package main

import (
	"fmt"
	"io"

	"github.com/beevik/etree"
)

type IUSFX interface {
	fmt.Stringer

	Document() *etree.Document
	USFXRootElement() (*etree.Element, error)
}

type USFX struct {
	doc *etree.Document
}

func (self *USFX) Document() *etree.Document {
	return self.doc
}

func (self *USFX) USFXRootElement() (*etree.Element, error) {
	str_path := "*[1]"
	path, err := etree.CompilePath(str_path)
	if err != nil {
		return nil, fmt.Errorf("Error compiling path: %s, %w", str_path, err)
	}
	elements := self.Document().FindElementsPathSeq(path)
	for element := range elements {
		if element.FullTag() == "usfx" {
			return element, nil
		}

		return nil, fmt.Errorf("first element in document is not <usfx>")
	}
	return nil, fmt.Errorf("no elements in document!")
}


func (self *USFX) String() string {
	return "TODO: this is a placeholder string for IUSFX.String"
}

func USFXFromReader(reader io.Reader) (IUSFX, error) {
	doc := etree.NewDocument()

	_, err := doc.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("[USFXFromReader] Error reading XML document (doc.ReadFrom): %w", err)
	}

	usfx := &USFX{doc: doc}
	return usfx, nil
}
