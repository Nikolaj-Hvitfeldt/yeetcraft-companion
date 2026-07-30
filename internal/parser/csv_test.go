package parser

import (
	"errors"
	"testing"
)

func TestTokenizeCSV(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    []string
		wantErr bool
	}{
		{"quoted name", `EVENT,guid,"Synthetic Name",nil`, []string{"EVENT", "guid", "Synthetic Name", "nil"}, false},
		{"quoted comma", `EVENT,"Synthetic, Name",tail`, []string{"EVENT", "Synthetic, Name", "tail"}, false},
		{"literal nil", `EVENT,nil`, []string{"EVENT", "nil"}, false},
		{"unknown trailing fields", `EVENT,a,b,c`, []string{"EVENT", "a", "b", "c"}, false},
		{"malformed quote", `EVENT,"unterminated`, nil, true},
		{"empty payload", ``, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TokenizeCSV(tt.payload)
			if tt.wantErr {
				if !errors.Is(err, ErrMalformedCSV) {
					t.Fatalf("error = %v, want ErrMalformedCSV", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("TokenizeCSV() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("fields = %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("fields[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
