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

	path := etree.MustCompilePath("[name()='w'][text()='water']")
	elements := el.FindElementsPathSeq(path)
	for element := range elements {
		parent_book := FindParentBook(element)
		parent_verse := FindVerse(element)

		fmt.Fprintf(out, "! %s (%s) [%s] %s: %s :: %s\n",
		// fmt.Fprintf(out, "! %s — %s: %s :: %s\n",
			element.FullTag(),
			parent_verse.SelectAttrValue("bcv", "%%%"),
			parent_book.SelectAttrValue("id", "???"),
			element.SelectAttrValue("s", "???"),
			element.Text(),
			parent_book.FullTag(),
		)

	}

	return nil
}

func FindParentBook(el *etree.Element) *etree.Element {
	if el == nil {
		return nil
	}

	if el.FullTag() == "book" {
		return el
	}

	return FindParentBook(el.Parent())
}


func FindVerse(el *etree.Element) *etree.Element {
	if el == nil {
		return nil
	}

	if el.FullTag() == "v" {
		return el
	}

	found := FindVerse(el.PrevSibling())
	if found == nil {
		if el.Parent().FullTag() == "q" {
			fmt.Println("Q?")
			return nil
		} else {
			fmt.Println("?Q")
			return nil
		}
	}

	return found
}
