package pgtest

import (
	"errors"
	"reflect"
	"testing"
)

type Person struct {
	Name string
	Age  int
}

var expErr = errors.New("expected error")

func TestGet(t *testing.T) {
	db := DB{
		Fatalf: t.Fatalf,
		Stack: []ResultMock{
			{"Get", &Person{"Bob", 45}, nil},
			{"Get", &Person{"Robert", 84}, nil},
			{"Get", nil, expErr},
		},
	}

	var p Person

	if err := db.Get(&p, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if p.Name != "Bob" || p.Age != 45 {
		t.Errorf("want Bob age 45, got %#v", p)
	}

	if err := db.Get(&p, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if p.Name != "Robert" || p.Age != 84 {
		t.Errorf("want Robert age 84, got %#v", p)
	}

	if err := db.Get(&p, ""); err != expErr {
		t.Errorf("want expErr, got %s (%#v)", err, p)
	}
}

func TestSelect(t *testing.T) {
	ps1 := []Person{{"Bob", 45}, {"Rob", 21}}
	ps2 := []Person{{"Jim", 84}}
	db := DB{
		Fatalf: t.Fatalf,
		Stack: []ResultMock{
			{"Select", &ps1, nil},
			{"Select", &ps2, nil},
			{"Select", nil, expErr},
		},
	}

	var ps []Person

	if err := db.Select(&ps, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(ps, ps1) {
		t.Errorf("want %#v, got %#v", ps1, ps)
	}

	if err := db.Select(&ps, ""); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(ps, ps2) {
		t.Errorf("want %#v, got %#v", ps1, ps)
	}

	if err := db.Select(&ps, ""); err != expErr {
		t.Errorf("want expErr, got %s (%#v)", err, ps)
	}
}

func TestInvalidMethod(t *testing.T) {
	var called bool
	fatalf := func(f string, args ...interface{}) {
		called = true
	}

	db := DB{
		Fatalf: fatalf,
		Stack: []ResultMock{
			{"Exec", nil, nil},
		},
	}

	var p Person
	if err := db.Get(&p, ""); err != ErrUnexpectedCall {
		t.Errorf("want %q, got %q", ErrUnexpectedCall, err)
	}
	if !called {
		t.Error("fatalf not called")
	}
}
