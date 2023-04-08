package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSizeToApproxHuman(t *testing.T) {
	assert.Equal(t, "1.0 KiB", sizeToApproxHuman(1024))
	assert.Equal(t, "1.1 KiB", sizeToApproxHuman(1126))
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"02.01.2006", time.Date(2006, 01, 02, 0, 0, 0, 0, time.Local)},
		{"02.01.2006 15:04", time.Date(2006, 01, 02, 15, 4, 0, 0, time.Local)},
		{"02.01.2006 15:04:05", time.Date(2006, 01, 02, 15, 4, 5, 0, time.Local)},
	}

	for _, test := range tests {
		got, err := parseTime(test.input)
		assert.NoError(t, err)

		assert.Equal(t, test.expected, got)
	}
}
