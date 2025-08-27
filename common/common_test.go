package common

import "testing"

func TestSliceHasString(t *testing.T) {
    tests := []struct {
        name     string
        slice    []string
        str      string
        expected bool
    }{
        {"string exists", []string{"apple", "banana", "cherry"}, "banana", true},
        {"string does not exist", []string{"apple", "banana", "cherry"}, "grape", false},
        {"empty slice", []string{}, "banana", false},
        {"single element slice, string exists", []string{"banana"}, "banana", true},
        {"single element slice, string does not exist", []string{"apple"}, "banana", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := SliceHasString(tt.slice, tt.str)
            if result != tt.expected {
                t.Errorf("SliceHasString(%v, %s) = %v; want %v", tt.slice, tt.str, result, tt.expected)
            }
        })
    }
}
