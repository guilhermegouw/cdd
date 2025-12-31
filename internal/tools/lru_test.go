package tools

import (
	"sync"
	"testing"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := NewLRUCache[string, int](3)

	// Test Put and Get
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("Get(a) = %v, %v; want 1, true", v, ok)
	}
	if v, ok := cache.Get("b"); !ok || v != 2 {
		t.Errorf("Get(b) = %v, %v; want 2, true", v, ok)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("Get(c) = %v, %v; want 3, true", v, ok)
	}

	// Test missing key
	if _, ok := cache.Get("d"); ok {
		t.Error("Get(d) should return false for missing key")
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	cache := NewLRUCache[string, int](2)

	cache.Put("a", 1)
	cache.Put("b", 2)

	// Access "a" to make it recently used
	cache.Get("a")

	// Add "c" - should evict "b" (least recently used)
	cache.Put("c", 3)

	if _, ok := cache.Get("b"); ok {
		t.Error("b should have been evicted")
	}
	if v, ok := cache.Get("a"); !ok || v != 1 {
		t.Errorf("a should still exist, got %v, %v", v, ok)
	}
	if v, ok := cache.Get("c"); !ok || v != 3 {
		t.Errorf("c should exist, got %v, %v", v, ok)
	}
}

func TestLRUCache_Update(t *testing.T) {
	cache := NewLRUCache[string, int](2)

	cache.Put("a", 1)
	cache.Put("a", 2) // Update

	if v, ok := cache.Get("a"); !ok || v != 2 {
		t.Errorf("Get(a) = %v, %v; want 2, true", v, ok)
	}
	if cache.Len() != 1 {
		t.Errorf("Len() = %d; want 1", cache.Len())
	}
}

func TestLRUCache_Delete(t *testing.T) {
	cache := NewLRUCache[string, int](3)

	cache.Put("a", 1)
	cache.Put("b", 2)

	if !cache.Delete("a") {
		t.Error("Delete(a) should return true")
	}
	if cache.Delete("a") {
		t.Error("Delete(a) again should return false")
	}
	if _, ok := cache.Get("a"); ok {
		t.Error("a should be deleted")
	}
	if cache.Len() != 1 {
		t.Errorf("Len() = %d; want 1", cache.Len())
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := NewLRUCache[string, int](3)

	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Len() = %d after Clear; want 0", cache.Len())
	}
	if _, ok := cache.Get("a"); ok {
		t.Error("a should not exist after Clear")
	}
}

func TestLRUCache_Metrics(t *testing.T) {
	cache := NewLRUCache[string, int](2)

	cache.Put("a", 1)
	cache.Get("a") // Hit
	cache.Get("a") // Hit
	cache.Get("b") // Miss
	cache.Get("c") // Miss

	hits, misses := cache.Metrics()
	if hits != 2 {
		t.Errorf("hits = %d; want 2", hits)
	}
	if misses != 2 {
		t.Errorf("misses = %d; want 2", misses)
	}

	hitRate := cache.HitRate()
	if hitRate != 50 {
		t.Errorf("HitRate() = %f; want 50", hitRate)
	}
}

func TestLRUCache_MinCapacity(t *testing.T) {
	cache := NewLRUCache[string, int](0) // Should be corrected to 1

	cache.Put("a", 1)
	cache.Put("b", 2) // Should evict "a"

	if cache.Len() != 1 {
		t.Errorf("Len() = %d; want 1", cache.Len())
	}
	if _, ok := cache.Get("a"); ok {
		t.Error("a should have been evicted")
	}
}

func TestLRUCache_ConcurrentAccess(t *testing.T) {
	cache := NewLRUCache[int, int](100)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Put(base*100+j, j)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Get(j)
			}
		}()
	}

	wg.Wait()

	// Just verify no panic and cache is bounded
	if cache.Len() > 100 {
		t.Errorf("Len() = %d; want <= 100", cache.Len())
	}
}

func TestLRUCache_LRUOrder(t *testing.T) {
	cache := NewLRUCache[string, int](3)

	// Add in order: a, b, c
	cache.Put("a", 1)
	cache.Put("b", 2)
	cache.Put("c", 3)

	// Access "a" to make it most recently used
	cache.Get("a")

	// Add "d" - should evict "b" (now least recently used)
	cache.Put("d", 4)

	if _, ok := cache.Get("b"); ok {
		t.Error("b should have been evicted (LRU)")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Error("a should still exist (was accessed)")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Error("c should still exist")
	}
	if _, ok := cache.Get("d"); !ok {
		t.Error("d should exist")
	}
}

func BenchmarkLRUCache_Put(b *testing.B) {
	cache := NewLRUCache[int, int](1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Put(i%10000, i)
	}
}

func BenchmarkLRUCache_Get(b *testing.B) {
	cache := NewLRUCache[int, int](1000)
	for i := 0; i < 1000; i++ {
		cache.Put(i, i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Get(i % 1000)
	}
}

func BenchmarkLRUCache_Mixed(b *testing.B) {
	cache := NewLRUCache[int, int](1000)
	for i := 0; i < 1000; i++ {
		cache.Put(i, i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			cache.Get(i % 1000)
		} else {
			cache.Put(i%10000, i)
		}
	}
}
