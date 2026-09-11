// Package javamap reproduces the iteration order of java.util.HashMap.
//
// GlobalExceptionHandler builds its field-error response in a `new HashMap<>()`
// and hands it to Jackson, which serialises a Map in the map's own iteration
// order. The key order a client sees is therefore HashMap's bucket order, not
// the order the handler inserted the entries. Go's map iteration is randomised
// by design, so that order has to be computed rather than inherited.
//
// The rule, from java.util.HashMap: the table holds a power-of-two number of
// buckets, an entry lands in bucket `spread(key.hashCode()) & (capacity-1)`
// where `spread(h) = h ^ (h >>> 16)`, iteration walks buckets in ascending
// index order, and entries that collide in one bucket are visited in insertion
// order.
package javamap

import (
	"bytes"
	"encoding/json"
	"sort"
	"unicode/utf16"
)

// defaultCapacity and loadFactor are HashMap's own defaults: a 16-bucket table
// that doubles once size exceeds 0.75 * capacity.
const (
	defaultCapacity = 16
	loadFactorNum   = 3
	loadFactorDen   = 4
)

// StringHashCode returns java.lang.String.hashCode for s.
//
// Java hashes UTF-16 code units, not code points, so a non-BMP character
// contributes its surrogate pair. The arithmetic is deliberately performed on
// uint32 and reinterpreted, matching Java's wrapping 32-bit int.
func StringHashCode(s string) int32 {
	var h uint32
	for _, unit := range utf16.Encode([]rune(s)) {
		h = h*31 + uint32(unit)
	}
	return int32(h)
}

// spread applies HashMap.hash: xor the high 16 bits down so that keys differing
// only above the table mask still separate into different buckets.
func spread(hash int32) uint32 {
	h := uint32(hash)
	return h ^ (h >> 16)
}

// capacityFor returns the table size a HashMap holds after size insertions,
// starting at 16 and doubling whenever size exceeds the load-factor threshold.
func capacityFor(size int) uint32 {
	capacity := uint32(defaultCapacity)
	for size > int(capacity)*loadFactorNum/loadFactorDen {
		capacity <<= 1
	}
	return capacity
}

// Bucket returns the table index key occupies in a HashMap holding size entries.
func Bucket(key string, size int) uint32 {
	return spread(StringHashCode(key)) & (capacityFor(size) - 1)
}

// Order returns keys rearranged into the order a HashMap holding exactly those
// keys would iterate them. The argument is taken in insertion order, which is
// what decides the outcome for keys that share a bucket.
func Order(keys []string) []string {
	ordered := make([]string, len(keys))
	copy(ordered, keys)
	sort.SliceStable(ordered, func(i, j int) bool {
		return Bucket(ordered[i], len(keys)) < Bucket(ordered[j], len(keys))
	})
	return ordered
}

// StringMap is the `Map<String, String>` GlobalExceptionHandler populates: it
// keeps insertion order for collision resolution, overwrites on a repeated key
// exactly as Map.put does, and marshals itself in HashMap iteration order.
type StringMap struct {
	inserted []string
	values   map[string]string
}

// NewStringMap returns an empty map, the equivalent of `new HashMap<>()`.
func NewStringMap() *StringMap {
	return &StringMap{values: map[string]string{}}
}

// Put stores value under key. A repeated key keeps its original bucket position
// and takes the new value, which is why a field carrying two violations shows
// only the last one written.
func (m *StringMap) Put(key, value string) {
	if _, seen := m.values[key]; !seen {
		m.inserted = append(m.inserted, key)
	}
	m.values[key] = value
}

// Len returns the number of entries.
func (m *StringMap) Len() int { return len(m.inserted) }

// Get returns the value stored under key, and whether it was present.
func (m *StringMap) Get(key string) (string, bool) {
	value, ok := m.values[key]
	return value, ok
}

// Keys returns the keys in HashMap iteration order.
func (m *StringMap) Keys() []string { return Order(m.inserted) }

// MarshalJSON writes the entries as a JSON object in HashMap iteration order,
// which is what Jackson does when handed a Map.
func (m *StringMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range m.Keys() {
		if i > 0 {
			buf.WriteByte(',')
		}
		encoded, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(encoded)
		buf.WriteByte(':')
		encoded, err = json.Marshal(m.values[key])
		if err != nil {
			return nil, err
		}
		buf.Write(encoded)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
