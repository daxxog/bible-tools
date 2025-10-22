package main

import (
	"fmt"
	"io"
	"os"
)

func _main(in io.Reader, out io.Writer) error {
	usfx, err := WEBUSFX()
	if err != nil {
		return fmt.Errorf("Error loading World English Bible USFX: %w", err)
	}

	err = RunUSFX(in, usfx, out)
	if err != nil {
		return fmt.Errorf("Error running USFX: %w", err)
	}

	return nil
}

func main() {
	err := _main(os.Stdin, os.Stdout)
	if err != nil {
		panic(err)
	}
}

