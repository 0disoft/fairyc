package contractstate

import "testing"

func TestByteValuesDoNotAliasAndEmptyValueDiffersFromDelete(t *testing.T) {
	t.Parallel()
	builder, err := NewLayoutBuilder("example.bytes")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := DeclareCell(builder, "payload", Bytes)
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(layout)
	if err != nil {
		t.Fatal(err)
	}
	limits := testLimits()
	input := []byte("abc")
	if _, err := Execute(store, layout, limits, func(transaction *Transaction) error {
		if err := payload.Set(transaction, input); err != nil {
			return err
		}
		input[0] = 'z'
		first, found, err := payload.Get(transaction)
		if err != nil {
			return err
		}
		if !found || string(first) != "abc" {
			t.Fatalf("pending value = %q, %v", first, found)
		}
		first[0] = 'y'
		second, _, err := payload.Get(transaction)
		if err != nil {
			return err
		}
		if string(second) != "abc" {
			t.Fatalf("pending value aliased read result: %q", second)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	dump := store.Dump()
	if len(dump) != 1 || string(dump[0].Value) != "abc" {
		t.Fatalf("Dump() = %+v", dump)
	}
	dump[0].Value[0] = 'x'
	if err := View(store, layout, limits, func(transaction *Transaction) error {
		value, found, err := payload.Get(transaction)
		if err != nil {
			return err
		}
		if !found || string(value) != "abc" {
			t.Fatalf("committed value aliased Dump(): %q, %v", value, found)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := Execute(store, layout, limits, func(transaction *Transaction) error {
		return payload.Set(transaction, []byte{})
	}); err != nil {
		t.Fatal(err)
	}
	if err := View(store, layout, limits, func(transaction *Transaction) error {
		value, found, err := payload.Get(transaction)
		if err != nil {
			return err
		}
		if !found || len(value) != 0 {
			t.Fatalf("empty stored value = %q, %v", value, found)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(store, layout, limits, func(transaction *Transaction) error {
		return payload.Delete(transaction)
	}); err != nil {
		t.Fatal(err)
	}
	if err := View(store, layout, limits, func(transaction *Transaction) error {
		_, found, err := payload.Get(transaction)
		if err != nil {
			return err
		}
		if found {
			t.Fatal("deleted value remains present")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
