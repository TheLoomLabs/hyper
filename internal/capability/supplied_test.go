package capability_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/capability"
)

// TestReadObject_ReadsTheObjectAndNotTheFile is the supplied response object as
// §12 closes it: the members the assembled object carries, at the shapes it
// holds them, in the order it builds them — whichever order and whichever
// spelling the caller wrote (§9, ADR-0108).
//
// The order and the lowering are the two that would be invisible without a
// case. A caller writing `body` before `host` gets `host` first because the
// object is the object; a caller writing `Content-Type` gets the member their
// path reaches, because a header name is case-insensitive on the wire and a
// path is exact.
func TestReadObject_ReadsTheObjectAndNotTheFile(t *testing.T) {
	object, err := capability.ReadObject(capability.CapabilityHTTP, []byte(`{
		"body": {"result": {"id": "abc"}},
		"headers": {"Content-Type": "application/json"},
		"host": "api.example.com",
		"tls": {"days_left": 29, "not_after": "2026-09-28T00:00:00Z"},
		"status": 201
	}`))
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}

	var names []string
	for _, member := range object {
		names = append(names, member.Name)
	}
	if want := []string{"host", "status", "headers", "body", "tls"}; strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("the object's members are %q, want §12's own order %q", names, want)
	}

	if status, carried := object.Lookup(capability.MemberStatus); !carried || status != 201 {
		t.Errorf("status is %#v, want the int 201: a status is an integer after a call and reads as one here", status)
	}
	headers, carried := object.Lookup(capability.MemberHeaders)
	if !carried {
		t.Fatal("the object carries no headers")
	}
	if lowered, isMapping := headers.(map[string]string); !isMapping || lowered["content-type"] != "application/json" {
		t.Errorf("headers is %#v, want the name lowered as it is off the wire", headers)
	}
	if tls, carried := object.Lookup(capability.MemberTLS); !carried {
		t.Error("the object carries no tls")
	} else if nested, isObject := tls.(capability.Object); !isObject {
		t.Errorf("tls is %#v, want the nested object §12 states", tls)
	} else if days, held := nested.Lookup(capability.MemberDaysLeft); !held || days != 29 {
		t.Errorf("tls.days_left is %#v, want the int 29", days)
	}
}

// TestReadObject_AMemberLeftOutIsAbsentAndNotNull is the ordinary absence rule
// arriving here: a member the file did not carry is not on the object at all,
// which is what keeps *resolved to nothing* and *resolved to null* two answers
// a projection tells apart (§7, §12).
func TestReadObject_AMemberLeftOutIsAbsentAndNotNull(t *testing.T) {
	object, err := capability.ReadObject(capability.CapabilityHTTP, []byte(`{"host": "api.example.com"}`))
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}
	if len(object) != 1 {
		t.Fatalf("the object carries %d members, want the one it was given: a response that answered nothing is host alone", len(object))
	}
	for _, name := range []string{capability.MemberStatus, capability.MemberBody, capability.MemberTLS} {
		if _, carried := object.Lookup(name); carried {
			t.Errorf("%s is on the object, and a member left out is absent rather than null", name)
		}
	}
}

// TestReadObject_TheShellObjectIsReadAgainstItsOwnFour is the second Capability,
// and the case that says the table is chosen by the Operation rather than by
// the caller: `stdout` is a member here and nowhere on the `http` object.
func TestReadObject_TheShellObjectIsReadAgainstItsOwnFour(t *testing.T) {
	object, err := capability.ReadObject(capability.CapabilityShell, []byte(`{"command": "[\"echo\",\"a b\"]", "exit_code": 0, "stdout": "a b", "stderr": ""}`))
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}
	if code, carried := object.Lookup(capability.MemberExitCode); !carried || code != 0 {
		t.Errorf("exit_code is %#v, want the int 0", code)
	}
	if _, err := capability.ReadObject(capability.CapabilityHTTP, []byte(`{"host": "h", "stdout": "a"}`)); err == nil {
		t.Error("the http object accepted stdout, and the two objects share no member")
	}
}

// TestReadObject_WhatItRefuses is every way a supplied object is one no call
// could have produced. Each of them is a path root that would resolve here and
// nowhere else, which is the whole reason the set is closed (ADR-0108).
func TestReadObject_WhatItRefuses(t *testing.T) {
	for _, c := range []struct {
		name, supplied, want string
	}{
		{
			name:     "a member no response object carries",
			supplied: `{"host": "h", "data": {}}`,
			want:     "no response object carries that member",
		},
		{
			name:     "a member no tls object carries",
			supplied: `{"host": "h", "tls": {"fingerprint": "aa"}}`,
			want:     "tls.fingerprint",
		},
		{
			name:     "the member that survives a call that answered nothing",
			supplied: `{"status": 200}`,
			want:     "the response object always carries it",
		},
		{
			name:     "a status that is not an integer",
			supplied: `{"host": "h", "status": "200"}`,
			want:     "the object holds an integer here",
		},
		{
			name:     "a status that is not whole",
			supplied: `{"host": "h", "status": 200.5}`,
			want:     "is not one",
		},
		{
			name:     "a headers mapping holding something other than a value",
			supplied: `{"host": "h", "headers": {"x": ["a", "b"]}}`,
			want:     "one value per name",
		},
		{
			name:     "a host that is not a string",
			supplied: `{"host": 200}`,
			want:     "the object holds a string here",
		},
		{
			name:     "a response that is not an object",
			supplied: `["host"]`,
			want:     "is not a JSON object",
		},
		{
			name:     "a response that is null",
			supplied: `null`,
			want:     "is null, and a response object is an object",
		},
		{
			name:     "two values where a response object is one",
			supplied: `{"host": "h"} {"host": "h"}`,
			want:     "a response object is one object",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := capability.ReadObject(capability.CapabilityHTTP, []byte(c.supplied))
			if err == nil {
				t.Fatalf("ReadObject read %s, want a fault naming %q", c.supplied, c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("ReadObject: %v, want a fault naming %q", err, c.want)
			}
		})
	}
}

// TestReadObject_ABodysNumbersAreItsLiterals is parseBody's rule read one
// source over: an integer past a float64's exact range is a Record identity on
// plenty of upstreams, and one that moved under a re-encode would be a
// different Record here than it is after a call (§7, §12).
func TestReadObject_ABodysNumbersAreItsLiterals(t *testing.T) {
	object, err := capability.ReadObject(capability.CapabilityHTTP, []byte(`{"host": "h", "body": {"id": 9007199254740993}}`))
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}
	body, _ := object.Lookup(capability.MemberBody)
	mapping, isMapping := body.(map[string]any)
	if !isMapping {
		t.Fatalf("body is %#v, want whatever the JSON parsed to", body)
	}
	number, isNumber := mapping["id"].(json.Number)
	if !isNumber || number.String() != "9007199254740993" {
		t.Errorf("body.id is %#v, want the literal it was written as", mapping["id"])
	}
}

// TestReadObject_ACapabilityWithNoObject is the one fault that is not about the
// file: there is one response object per Capability and no third (§12).
func TestReadObject_ACapabilityWithNoObject(t *testing.T) {
	if _, err := capability.ReadObject("dns", []byte(`{}`)); err == nil {
		t.Error("ReadObject answered for a Capability that has no response object")
	}
}

// TestReadObject_TwoSpellingsOfOneHeaderNameAreRefused is the one shape a
// supplied object can take that no call could: the wire joins repeated field
// lines into one value before the object is built (§12), so a file carrying
// both spellings is asking which of two values a path resolves to — a question
// a Go map answers differently on different runs.
func TestReadObject_TwoSpellingsOfOneHeaderNameAreRefused(t *testing.T) {
	_, err := capability.ReadObject(capability.CapabilityHTTP,
		[]byte(`{"host": "h", "headers": {"Content-Type": "application/json", "content-type": "text/html"}}`))
	if err == nil {
		t.Fatal("ReadObject collapsed two spellings of one header name, and which value survived would be a fact about a map's iteration order")
	}
	for _, want := range []string{"Content-Type", "content-type", "one value per name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ReadObject: %v, want a fault naming %q", err, want)
		}
	}
}
