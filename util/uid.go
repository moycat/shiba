package util

import (
	"encoding/base32"
	"encoding/binary"
	"math/rand"
	"strings"
	"time"
)

var counter uint16

func init() {
	rand.Seed(time.Now().UnixNano())
	counter = uint16(rand.Int() % 65536)
}

// NewUID tries to return a locally unique ID.
// Don't use it concurrently. Not thread safe.
func NewUID() string {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(time.Now().Unix()))
	counter++
	b = append(b, byte(counter>>8), byte(counter))
	return strings.ToLower(base32.StdEncoding.EncodeToString(b))[:10]
}
