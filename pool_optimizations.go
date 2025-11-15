package vecto

import (
	"bytes"
	"sync"
)

var (
	headerMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]string, 8)
		},
	}

	paramMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]any, 4)
		},
	}

	bufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}
)

func getHeaderMap() map[string]string {
	return headerMapPool.Get().(map[string]string)
}

func putHeaderMap(m map[string]string) {
	if m == nil {
		return
	}
	for k := range m {
		delete(m, k)
	}
	if len(m) > 64 {
		return
	}
	headerMapPool.Put(m)
}

func getParamMap() map[string]any {
	return paramMapPool.Get().(map[string]any)
}

func putParamMap(m map[string]any) {
	if m == nil {
		return
	}
	for k := range m {
		delete(m, k)
	}
	if len(m) > 64 {
		return
	}
	paramMapPool.Put(m)
}

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(b *bytes.Buffer) {
	if b == nil {
		return
	}
	b.Reset()
	if b.Cap() > 64*1024 {
		return
	}
	bufferPool.Put(b)
}

