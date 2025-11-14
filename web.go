package main

import (
	"fmt"
	"os"
)

const WEB_USFX_PATH = "./opt/usfx/eng-web/eng-web_usfx.xml"

func WEBUSFX() (IUSFX, error) {
	file, err := os.Open(WEB_USFX_PATH)
	if err != nil {
		return nil, fmt.Errorf("[WEB] Error loading USFX from file(%s): %w", WEB_USFX_PATH, err)
	}

	usfx, err := USFXFromReader(file)
	if err != nil {
		return nil, fmt.Errorf("[WEB] Error loading USFX from reader: %w", err)
	}

	return usfx, nil
}
