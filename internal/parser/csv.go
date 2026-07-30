package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// TokenizeCSV parses exactly one CSV record and preserves every field.
func TokenizeCSV(payload string) ([]string, error) {
	if payload == "" {
		return nil, fmt.Errorf("%w: empty payload", ErrMalformedCSV)
	}

	reader := csv.NewReader(strings.NewReader(payload))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = false

	fields, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedCSV, err)
	}
	if _, err := reader.Read(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%w: multiple records in one payload", ErrMalformedCSV)
		}
		return nil, fmt.Errorf("%w: %v", ErrMalformedCSV, err)
	}
	return fields, nil
}
