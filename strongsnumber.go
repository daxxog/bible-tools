package main

import (
	"fmt"
	"strconv"
)

// strongsNumber implements IStrongsNumber.
type strongsNumber struct {
	strongs_type string
	value        uint16
}

func (s *strongsNumber) Type() string  { return s.strongs_type }
func (s *strongsNumber) Value() uint16 { return s.value }
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

