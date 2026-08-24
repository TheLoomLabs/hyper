package revision

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// git's `cat-file --batch` answer, read as the records it is (§7, §8).
//
// Two readers in this package ask git for objects that way and one of them asks
// for many at once, so the record itself is read here rather than at either: the
// protocol is git's — `<object> SP <type> SP <size> LF`, the content, and one LF
// — and a second reading of it is a second place the trailing LF can be
// miscounted.
//
// **The content is read by the size git states** and never by scanning for a
// separator, because an artefact may hold any byte at all, a newline among them.

// batchRecord is one object as that answer carries it: the header's three
// fields, the content, and what is left of the answer behind it.
//
// A record git could not produce carries `missing` where the type stands and no
// content at all, which is git's own word for it with the name echoed ahead. It
// is a record like any other here — whether an absent object is a failure is the
// caller's question, and the two callers answer it differently (ADR-0071).
type batchRecord struct {
	fields  []string
	content []byte
	rest    []byte
}

// missing reports whether git answered that it cannot produce the object. It
// covers every way that answer comes back at once — an object the clone does
// not hold, a commit it does not hold, a path that commit's tree does not carry.
func (r batchRecord) missing() bool {
	return len(r.fields) == 2 && r.fields[1] == "missing"
}

// kind is the object's type, and "" on a record git could not produce.
func (r batchRecord) kind() string {
	if len(r.fields) != 3 {
		return ""
	}
	return r.fields[1]
}

// object is the object's own id, and "" on a record git could not produce.
func (r batchRecord) object() string {
	if len(r.fields) != 3 {
		return ""
	}
	return r.fields[0]
}

// readBatchRecord reads the first record off an answer, naming the object it
// was asked about in whatever it cannot read.
func readBatchRecord(answer []byte, named string) (batchRecord, error) {
	header, body, split := bytes.Cut(answer, []byte("\n"))
	if !split {
		return batchRecord{}, fmt.Errorf("git cat-file answered no header for %s", named)
	}
	fields := strings.Fields(string(header))
	if len(fields) == 2 && fields[1] == "missing" {
		return batchRecord{fields: fields, rest: body}, nil
	}
	if len(fields) != 3 {
		return batchRecord{}, fmt.Errorf("git cat-file wrote %q for %s, which is neither <object> <type> <size> nor a missing object", header, named)
	}
	size, err := strconv.Atoi(fields[2])
	if err != nil || size < 0 || size+1 > len(body) {
		return batchRecord{}, fmt.Errorf("git cat-file wrote %q for %s, which is not a size this answer carries", header, named)
	}
	return batchRecord{fields: fields, content: body[:size], rest: body[size+1:]}, nil
}
