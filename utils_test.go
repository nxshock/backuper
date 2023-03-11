package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSizeToApproxHuman(t *testing.T) {
	assert.Equal(t, "1.0 KiB", sizeToApproxHuman(1024))
	assert.Equal(t, "1.1 KiB", sizeToApproxHuman(1126))
}
