package envconf

import (
	"reflect"
	"testing"
)

func TestLoadInvalidDest(t *testing.T) {
	input := map[string]string{
		"a": "321",
	}

	var a int
	if err := Load(a, input); err == nil {
		t.Error("expected error")
	}
	if err := Load(&a, input); err == nil {
		t.Error("expected error")
	}

	var b bool
	if err := Load(b, input); err == nil {
		t.Error("expected error")
	}
	if err := Load(&b, input); err == nil {
		t.Error("expected error")
	}

	var c string
	if err := Load(c, input); err == nil {
		t.Error("expected error")
	}
	if err := Load(&c, input); err == nil {
		t.Error("expected error")
	}

	var d struct{}
	if err := Load(d, input); err == nil {
		t.Error("expected error")
	}
}

func TestLoadString(t *testing.T) {
	var c struct {
		First  string
		Second string
	}
	in := map[string]string{
		"FIRST":  "foo",
		"SECOND": "bar",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if c.First != in["FIRST"] || c.Second != in["SECOND"] {
		t.Errorf("invalid conf: %+v", c)
	}
}

func TestLoadStringSlice(t *testing.T) {
	var c struct {
		First  []string
		Second []string
	}
	in := map[string]string{
		"FIRST":  "foo;baz",
		"SECOND": "a;b;c",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if !reflect.DeepEqual(c.First, []string{"foo", "baz"}) {
		t.Errorf("first: %v", c.First)
	}
	if !reflect.DeepEqual(c.Second, []string{"a", "b", "c"}) {
		t.Errorf("second: %v", c.Second)
	}
}

func TestLoadInt(t *testing.T) {
	var c struct {
		Int   int
		Int8  int8
		Int16 int16
		Int32 int32
		Int64 int64
	}
	in := map[string]string{
		"INT":   "1",
		"INT8":  "2",
		"INT16": "3",
		"INT32": "4",
		"INT64": "5",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if c.Int != 1 || c.Int8 != 2 || c.Int16 != 3 || c.Int32 != 4 || c.Int64 != 5 {
		t.Errorf("invalid conf: %+v", c)
	}
}

func TestLoadIntSlice(t *testing.T) {
	var c struct {
		Int   []int
		Int8  []int8
		Int16 []int16
		Int32 []int32
		Int64 []int64
	}
	in := map[string]string{
		"INT":   "1;1",
		"INT8":  "2;2",
		"INT16": "3;3",
		"INT32": "4;4",
		"INT64": "5;5",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if !reflect.DeepEqual(c.Int, []int{1, 1}) {
		t.Errorf("unexpected Int value: %+v", c.Int)
	}
	if !reflect.DeepEqual(c.Int8, []int8{2, 2}) {
		t.Errorf("unexpected Int8 value: %+v", c.Int8)
	}
	if !reflect.DeepEqual(c.Int16, []int16{3, 3}) {
		t.Errorf("unexpected Int16 value: %+v", c.Int16)
	}
	if !reflect.DeepEqual(c.Int32, []int32{4, 4}) {
		t.Errorf("unexpected Int32 value: %+v", c.Int32)
	}
	if !reflect.DeepEqual(c.Int64, []int64{5, 5}) {
		t.Errorf("unexpected Int64 value: %+v", c.Int64)
	}
}

func TestBool(t *testing.T) {
	var c struct {
		A bool
		B bool
		C bool
		D bool
		E bool
		F bool
	}
	in := map[string]string{
		"A": "t",
		"B": "true",
		"C": "1",
		"D": "f",
		"E": "false",
		"F": "0",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if !c.A || !c.B || !c.C || c.D || c.E || c.F {
		t.Errorf("invalid conf: %+v", c)
	}
}

func TestBoolSlice(t *testing.T) {
	var c struct {
		A []bool
		B []bool
	}
	in := map[string]string{
		"A": "t;true;1",
		"B": "f;false;0",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if !reflect.DeepEqual(c.A, []bool{true, true, true}) {
		t.Errorf("invalid A value: %+v", c.A)
	}
	if !reflect.DeepEqual(c.B, []bool{false, false, false}) {
		t.Errorf("invalid B value: %+v", c.B)
	}
}

func TestFloat(t *testing.T) {
	var c struct {
		A float32
		B float64
	}
	in := map[string]string{
		"A": "1.1",
		"B": "2.2",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if c.A != 1.1 || c.B != 2.2 {
		t.Errorf("invalid conf: %+v", c)
	}
}

func TestFloatSlice(t *testing.T) {
	var c struct {
		A []float32
		B []float64
	}
	in := map[string]string{
		"A": "1.1;1.2",
		"B": "2.2;2.3",
	}
	if err := Load(&c, in); err != nil {
		t.Fatalf("cannot load configuration: %s", err)
	}
	if !reflect.DeepEqual(c.A, []float32{1.1, 1.2}) {
		t.Errorf("invalid A value: %+v", c.A)
	}
	if !reflect.DeepEqual(c.B, []float64{2.2, 2.3}) {
		t.Errorf("invalid B value: %+v", c.B)
	}
}

func TestConvertName(t *testing.T) {
	var testCases = []struct {
		input string
		want  string
	}{
		{"FooBar", "FOO_BAR"},
		{"HTTPPort", "HTTP_PORT"},
		{"ServerFullAddress", "SERVER_FULL_ADDRESS"},
	}
	for i, tc := range testCases {
		got := convertName(tc.input)
		if got != tc.want {
			t.Errorf("%d: want %q, got %q", i, tc.want, got)
		}
	}
}
