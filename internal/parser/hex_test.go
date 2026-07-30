package parser

import "testing"

func TestParseHexUint32(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		want   uint32
		wantOK bool
	}{
		{name: "zero", token: "0x0", want: 0, wantOK: true},
		{name: "uppercase prefix", token: "0X1", want: 1, wantOK: true},
		{name: "maximum", token: "0xffffffff", want: 0xffffffff, wantOK: true},
		{name: "decimal-looking", token: "10"},
		{name: "octal-looking", token: "010"},
		{name: "missing prefix", token: "ff"},
		{name: "empty suffix", token: "0x"},
		{name: "malformed", token: "0xnot-hex"},
		{name: "overflow", token: "0x100000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseHexUint32(tt.token)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("parseHexUint32(%q) = %#x, %v; want %#x, %v", tt.token, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
