package vecto

import (
	"bytes"
	"sync"
)

var (
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

func getStringBuilder() *bytes.Buffer {
	sb := stringBuilderPool.Get().(*bytes.Buffer)
	sb.Reset()
	return sb
}

func putStringBuilder(sb *bytes.Buffer) {
	if sb.Cap() > 4*1024 {
		return
	}
	stringBuilderPool.Put(sb)
}
