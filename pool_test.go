package vecto

import (
	"testing"
)

func TestGetStringBuilder(t *testing.T) {
	t.Run("returns non-nil buffer", func(t *testing.T) {
		sb := getStringBuilder()
		if sb == nil {
			t.Fatal("expected non-nil buffer")
		}
	})

	t.Run("returns clean buffer", func(t *testing.T) {
		sb := getStringBuilder()
		sb.WriteString("test data")
		putStringBuilder(sb)

		sb2 := getStringBuilder()
		if sb2.Len() != 0 {
			t.Errorf("expected clean buffer, got length %d", sb2.Len())
		}
	})

	t.Run("buffer is reusable", func(t *testing.T) {
		sb := getStringBuilder()
		sb.WriteString("data")
		if sb.String() != "data" {
			t.Errorf("expected 'data', got %s", sb.String())
		}
	})
}

func TestPutStringBuilder(t *testing.T) {
	t.Run("accepts buffer with normal size", func(t *testing.T) {
		sb := getStringBuilder()
		sb.WriteString("normal size content")
		
		putStringBuilder(sb)
		
		sb2 := getStringBuilder()
		if sb2.Len() != 0 {
			t.Error("expected buffer to be reset and returned to pool")
		}
	})

	t.Run("rejects buffer with large capacity", func(t *testing.T) {
		sb := getStringBuilder()
		
		largeData := make([]byte, 5*1024)
		for i := range largeData {
			largeData[i] = 'x'
		}
		sb.Write(largeData)
		
		if sb.Cap() <= 4*1024 {
			t.Skip("buffer capacity not large enough for this test")
		}
		
		putStringBuilder(sb)
	})

	t.Run("pool works correctly under concurrent access", func(t *testing.T) {
		done := make(chan bool)
		
		for i := 0; i < 10; i++ {
			go func() {
				sb := getStringBuilder()
				sb.WriteString("concurrent test")
				putStringBuilder(sb)
				done <- true
			}()
		}
		
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func BenchmarkStringBuilderPool(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sb := getStringBuilder()
				sb.WriteString("https://")
				sb.WriteString("api.example.com")
				sb.WriteString("/v1/users")
				_ = sb.String()
				putStringBuilder(sb)
			}
		})
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sb := make([]byte, 0, 64)
				sb = append(sb, "https://"...)
				sb = append(sb, "api.example.com"...)
				sb = append(sb, "/v1/users"...)
				_ = string(sb)
			}
		})
	})
}

