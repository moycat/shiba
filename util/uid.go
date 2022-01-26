package util

import (
	"encoding/base32"
	"encoding/binary"
	"math/rand"
	"strings"
	"time"
)

var counter uint8

func init() {
	rand.Seed(time.Now().UnixNano())
	counter = uint8(rand.Int() % 255)
}

// NewUID tries to return a locally unique ID.
// Don't use it concurrently. Not thread safe.
func NewUID() string {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(time.Now().Unix()))
	counter++
	b = append(b, counter)
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))
}
