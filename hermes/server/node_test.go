package server

import (
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestNodeReplication(t *testing.T) {
	a, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("cannot listen: %s", err)
	}
	defer a.Close()

	b, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("cannot listen: %s", err)
	}
	defer b.Close()

	errc := make(chan error)

	go func() {
		na := NewNode()
		if err := na.Serve(a); err != nil {
			errc <- err
		}
	}()

	nb := NewNode()
	if err := nb.FollowAddr(a.Addr().String()); err != nil {
		t.Fatalf("cannot follow a: %s", err)
	}

	go func() {
		if err := nb.Serve(b); err != nil {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		t.Fatalf("node error: %s", err)
	default:
	}

	ca, err := net.Dial("tcp", a.Addr().String())
	if err != nil {
		t.Fatalf("cannot connect client to a: %s", err)
	}
	defer ca.Close()
	if _, err := fmt.Fprintln(ca, "v1"); err != nil {
		t.Fatalf("cannot write to a: %s", err)
	}

	cb, err := net.Dial("tcp", b.Addr().String())
	if err != nil {
		t.Fatalf("cannot connect client to b: %s", err)
	}
	defer cb.Close()
	if _, err := fmt.Fprintln(cb, "v1"); err != nil {
		t.Fatalf("cannot write to a: %s", err)
	}

	// give nodes some time to update subscriptions
	time.Sleep(50 * time.Millisecond)

	if _, err := fmt.Fprintln(ca, "ca hi"); err != nil {
		t.Fatalf("cannot write to a: %s", err)
	}
	buf := make([]byte, 1024)
	if n, err := cb.Read(buf); err != nil {
		t.Fatalf("cannot read from b: %s", err)
	} else {
		if got, want := string(buf[:n]), "ca hi\n"; got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}
