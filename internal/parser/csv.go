package parser

import (
	"encoding/csv"
	"io"
	"strings"
)

// TokenizeCSV parses exactly one CSV record and preserves every field.
func TokenizeCSV(payload string) ([]string, error) {
	if payload == "" {
		return nil, ErrMalformedCSV
	}

	reader := csv.NewReader(strings.NewReader(payload))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = false

	fields, err := reader.Read()
	if err != nil {
		return nil, ErrMalformedCSV
	}
	if _, err := reader.Read(); err != io.EOF {
		return nil, ErrMalformedCSV
	}
	return fields, nil
}
