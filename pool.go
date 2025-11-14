package vecto

import (
	"bytes"
	"sync"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		return
	}
	bufferPool.Put(buf)
}

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

