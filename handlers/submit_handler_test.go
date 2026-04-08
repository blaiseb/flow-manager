package handlers

import (
	"reflect"
	"testing"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
		wantErr  bool
	}{
		{"80", []int{80}, false},
		{"80, 443", []int{80, 443}, false},
		{"80-82", []int{80, 81, 82}, false},
		{"80, 8080-8082, 443", []int{80, 8080, 8081, 8082, 443}, false},
		{"", nil, true},
		{"abc", nil, true},
		{"80-70", nil, true},
		{"80-80-80", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePorts(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePorts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parsePorts() = %v, want %v", got, tt.expected)
			}
		})
	}
}
