package parser

import (
	"errors"
	"strings"
	"testing"
)

func TestLineReaderChunksAndTerminators(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		want   []string
	}{
		{"multiple lines", []string{"one\ntwo\n"}, []string{"one", "two"}},
		{"split line", []string{"o", "n", "e\n"}, []string{"one"}},
		{"CRLF", []string{"one\r\ntwo\r\n"}, []string{"one", "two"}},
		{"split CRLF", []string{"one\r", "\ntwo\r", "\n"}, []string{"one", "two"}},
		{"empty lines", []string{"\n\r\nvalue\n"}, []string{"", "", "value"}},
		{"newline boundary", []string{"one", "\n", "two\n"}, []string{"one", "two"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewLineReader(DefaultMaxLineSize)
			var got []string
			for _, chunk := range tt.chunks {
				lines, err := reader.Write([]byte(chunk))
				if err != nil {
					t.Fatalf("Write() error = %v", err)
				}
				got = append(got, lines...)
			}
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("lines = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLineReaderFinalize(t *testing.T) {
	reader := NewLineReader(10)
	if _, err := reader.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	tail, hasTail, err := reader.Finalize()
	if err != nil || !hasTail || tail != "partial" {
		t.Fatalf("Finalize() = %q, %v, %v", tail, hasTail, err)
	}

	complete := NewLineReader(10)
	if _, err := complete.Write([]byte("done\n")); err != nil {
		t.Fatal(err)
	}
	if tail, hasTail, err := complete.Finalize(); err != nil || hasTail || tail != "" {
		t.Fatalf("complete Finalize() = %q, %v, %v", tail, hasTail, err)
	}
}

func TestLineReaderMaximumContentBytes(t *testing.T) {
	exact := strings.Repeat("x", DefaultMaxLineSize)
	tests := []struct {
		name   string
		chunks []string
	}{
		{"LF excluded", []string{exact + "\n"}},
		{"CRLF excluded", []string{exact + "\r\n"}},
		{"split CRLF excluded", []string{exact + "\r", "\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewLineReader(DefaultMaxLineSize)
			var lines []string
			for _, chunk := range tt.chunks {
				got, err := reader.Write([]byte(chunk))
				if err != nil {
					t.Fatalf("Write() error = %v", err)
				}
				lines = append(lines, got...)
			}
			if len(lines) != 1 || len(lines[0]) != DefaultMaxLineSize {
				t.Fatalf("got %d lines with first length %d", len(lines), len(lines[0]))
			}
		})
	}
}

func TestLineReaderOversizedIsTerminalAndPreservesCause(t *testing.T) {
	reader := NewLineReader(DefaultMaxLineSize)
	chunk := "before\n" + strings.Repeat("x", DefaultMaxLineSize+1) + "\nafter\n"
	lines, err := reader.Write([]byte(chunk))
	if len(lines) != 1 || lines[0] != "before" {
		t.Fatalf("lines = %#v, want prior complete line only", lines)
	}
	if !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("error = %v, want ErrLineTooLong", err)
	}
	if !reader.Failed() {
		t.Fatal("reader should be terminal")
	}

	if _, err := reader.Write([]byte("new\n")); !errors.Is(err, ErrReaderTerminal) || !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("terminal Write error = %v, want terminal and original cause", err)
	}
	if _, _, err := reader.Finalize(); !errors.Is(err, ErrReaderTerminal) || !errors.Is(err, ErrLineTooLong) {
		t.Fatalf("terminal Finalize error = %v, want terminal and original cause", err)
	}
}
