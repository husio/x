package log

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	now := time.Time{}
	defer testWithTime(now)()
	read, close := catchLoggerOut()
	defer close()

	for tname, tc := range cases {
		tc.Log(tc.Msg, tc.Keyvals...)
		got := make(map[string]string)
		err := json.Unmarshal(read(), &got)
		if err != nil {
			t.Errorf("%s: cannot unmarshal json: %s", tname, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.Want) {
			t.Errorf("%s failed\ngot: %#v\nexpected:%#v",
				tname, got, tc.Want)
		}
	}
}

// This is declared outside the test case to ensure the line
// number stays constant.
var cases = map[string]struct {
	Log     func(msg string, keyvals ...string)
	Msg     string
	Keyvals []string
	Want    map[string]string
}{
	"no_pairs_debug": {
		Log:     Debug,
		Msg:     "test error",
		Keyvals: []string{},
		Want: map[string]string{
			"msg":   "test error",
			"date":  time.Time{}.UTC().Format(time.RFC3339),
			"file":  "log_test.go:18",
			"level": "DEBUG",
		},
	},
	"with_pairs_debug": {
		Log:     Debug,
		Msg:     "test error",
		Keyvals: []string{"key1", "val1", "key2", "val2", "key3"},
		Want: map[string]string{
			"msg":   "test error",
			"date":  time.Time{}.UTC().Format(time.RFC3339),
			"file":  "log_test.go:18",
			"level": "DEBUG",
			"key1":  "val1",
			"key2":  "val2",
			"key3":  "",
		},
	},
	"no_pairs_err": {
		Log:     Error,
		Msg:     "test error",
		Keyvals: []string{},
		Want: map[string]string{
			"msg":   "test error",
			"date":  time.Time{}.UTC().Format(time.RFC3339),
			"file":  "log_test.go:18",
			"level": "ERROR",
		},
	},
	"with_pairs_err": {
		Log:     Error,
		Msg:     "test error",
		Keyvals: []string{"key1", "val1", "key2", "val2", "key3"},
		Want: map[string]string{
			"msg":   "test error",
			"date":  time.Time{}.UTC().Format(time.RFC3339),
			"file":  "log_test.go:18",
			"level": "ERROR",
			"key1":  "val1",
			"key2":  "val2",
			"key3":  "",
		},
	},
}

func testWithTime(now time.Time) func() {
	currentTime = func() time.Time { return now }
	return func() {
		currentTime = time.Now
	}
}

func catchLoggerOut() (func() []byte, func()) {
	write := root.write
	buf := &bytes.Buffer{}
	root.write = json.NewEncoder(buf).Encode

	read := func() []byte {
		b := buf.Bytes()
		buf.Reset()
		return b
	}
	close := func() {
		root.write = write
	}
	return read, close
}
