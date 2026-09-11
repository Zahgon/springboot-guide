package javamap

import (
	"encoding/json"
	"strings"
	"testing"
)

// Every expected value below was produced by running the corresponding
// expression on OpenJDK 11 and copying the result, so these tests compare this
// package against java.util.HashMap rather than against a second derivation of
// the same formula.

func TestStringHashCodeMatchesJava(t *testing.T) {
	for _, testCase := range []struct {
		input string
		want  int32
	}{
		{"", 0},
		{"a", 97},
		{"Aa", 2112},
		{"BB", 2112}, // the classic collision: same hash, different string
		{"classId", 853619891},
		{"name", 3373707},
		{"sex", 113766},
		{"region", -934795532},
		{"phoneNumber", -1192969641},
		{"group", 98629247},
		{"polygenelubricants", -2147483648}, // hashes to exactly Integer.MIN_VALUE
		{"一二三", 19832573},
		{"😀", 1772899},
	} {
		if got := StringHashCode(testCase.input); got != testCase.want {
			t.Errorf("StringHashCode(%q) = %d, want %d", testCase.input, got, testCase.want)
		}
	}
}

func TestBucketPlacesPersonRequestFields(t *testing.T) {
	// The five field names, and the buckets a 16-entry table puts them in. This
	// is what fixes the JSON key order of a validation failure.
	for _, testCase := range []struct {
		key  string
		want uint32
	}{
		{"classId", 2},
		{"phoneNumber", 3},
		{"sex", 7},
		{"name", 8},
		{"region", 12},
	} {
		if got := Bucket(testCase.key, 5); got != testCase.want {
			t.Errorf("Bucket(%q) = %d, want %d", testCase.key, got, testCase.want)
		}
	}
}

func TestOrderIsIndependentOfInsertionOrder(t *testing.T) {
	want := []string{"classId", "phoneNumber", "sex", "name", "region"}
	for _, insertion := range [][]string{
		{"classId", "name", "sex", "region", "phoneNumber"},
		{"region", "phoneNumber", "name", "sex", "classId"},
		{"sex", "classId", "region", "name", "phoneNumber"},
	} {
		got := Order(insertion)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("Order(%v) = %v, want %v", insertion, got, want)
		}
	}
}

func TestOrderOfTheFourFieldSubset(t *testing.T) {
	// The response the ported controller test asserts on: classId is valid, so
	// only the other four appear, and they keep their relative order.
	got := Order([]string{"name", "sex", "region", "phoneNumber"})
	want := "phoneNumber,sex,name,region"
	if strings.Join(got, ",") != want {
		t.Errorf("Order = %v, want %s", got, want)
	}
}

func TestOrderResizesPastTheLoadFactor(t *testing.T) {
	// A 13th entry doubles the table to 32, which moves keys to new buckets. A
	// fixed capacity of 16 would report the wrong order here.
	if got, want := Bucket("region", 5), uint32(12); got != want {
		t.Errorf("Bucket(region, size 5) = %d, want %d", got, want)
	}
	if got, want := Bucket("region", 13), uint32(28); got != want {
		t.Errorf("Bucket(region, size 13) = %d, want %d", got, want)
	}

	// `new HashMap<>()` with these thirteen keys iterates in exactly this order
	// on OpenJDK 11. A table pinned at 16 buckets gets it wrong.
	thirteen := []string{"classId", "name", "sex", "region", "phoneNumber",
		"a", "b", "c", "d", "e", "f", "g", "h"}
	got := strings.Join(Order(thirteen), ",")
	want := "a,b,c,d,e,f,sex,g,h,classId,phoneNumber,name,region"
	if got != want {
		t.Errorf("Order(13 keys) = %s, want %s", got, want)
	}
}

func TestOrderKeepsInsertionOrderWithinACollidingBucket(t *testing.T) {
	// "Aa" and "BB" have the same hashCode (2112) and so share a bucket; a
	// HashMap visits them in the order they were inserted. OpenJDK 11 prints
	// [BB, Aa, sex] for this insertion sequence.
	got := strings.Join(Order([]string{"BB", "Aa", "sex"}), ",")
	if want := "BB,Aa,sex"; got != want {
		t.Errorf("Order = %s, want %s", got, want)
	}
	got = strings.Join(Order([]string{"Aa", "BB", "sex"}), ",")
	if want := "Aa,BB,sex"; got != want {
		t.Errorf("Order = %s, want %s", got, want)
	}
}

func TestStringMapMarshalsInHashMapOrder(t *testing.T) {
	errors := NewStringMap()
	// Inserted in validation order, which is field-declaration order.
	errors.Put("name", "name 不能为空")
	errors.Put("sex", "sex 值不在可选范围")
	errors.Put("region", "Region 值不在可选范围内")
	errors.Put("phoneNumber", "phoneNumber 格式不正确")

	encoded, err := json.Marshal(errors)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"phoneNumber":"phoneNumber 格式不正确","sex":"sex 值不在可选范围",` +
		`"name":"name 不能为空","region":"Region 值不在可选范围内"}`
	if string(encoded) != want {
		t.Errorf("Marshal = %s, want %s", encoded, want)
	}
}

func TestStringMapPutKeepsLastValueAtTheOriginalPosition(t *testing.T) {
	// Map.put overwrites; the handler's `errors.put(field, message)` is why a
	// field carrying two violations reports only the last one written.
	errors := NewStringMap()
	errors.Put("name", "first")
	errors.Put("sex", "sex 不能为空")
	errors.Put("name", "second")

	if got, _ := errors.Get("name"); got != "second" {
		t.Errorf("Get(name) = %q, want %q", got, "second")
	}
	if errors.Len() != 2 {
		t.Errorf("Len = %d, want 2", errors.Len())
	}
	encoded, _ := json.Marshal(errors)
	if want := `{"sex":"sex 不能为空","name":"second"}`; string(encoded) != want {
		t.Errorf("Marshal = %s, want %s", encoded, want)
	}
}

func TestStringMapMarshalsEmpty(t *testing.T) {
	encoded, err := json.Marshal(NewStringMap())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(encoded) != "{}" {
		t.Errorf("Marshal = %s, want {}", encoded)
	}
}
