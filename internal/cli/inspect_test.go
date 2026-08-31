package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The two listings over the record, held to saying where the record is (§9,
// ADR-0113, issue #233).
//
// The corpus is what says what each page looks like; what these hold is the one
// claim a golden could lose to a regeneration nobody read — that the sentence
// stands on **every** page these two commands write, and not only on the empty
// one it used to stand on.

// TestInspectionListings_SayWhereTheRecordIsOnEveryPage is the fault the ticket
// found, held at the grain it was found at.
//
// A session that had run two Procedures was asked what its repository's account
// amounted to. It read the account back correctly off `changes`, went looking
// for where it was held, found no `.hyper/`, no `store/` and a clean `git
// status`, and told a human that a clone would get the Procedure and not the
// history. That is false of every Store there has ever been, and no surface it
// was allowed to call would have said so: `runs` and `records` named the branch
// when they had nothing to list and stopped naming it the moment they did.
//
// The four cases are the four pages between them: rows, and the empty page each
// writes, since a listing that found nothing is the one most easily read as
// *there is no record*.
func TestInspectionListings_SayWhereTheRecordIsOnEveryPage(t *testing.T) {
	entry := runRow{Type: "run", ID: "01991ea6-b118-7c93-8d41-6b2f7ae05c19", Procedure: "fleet-rollout", Targets: []string{"local"}}
	version := recordRow{
		Type:       "record",
		Key:        recordKey{Target: "local", Definition: "uptime", Name: "status.hyper.dev"},
		Ordinal:    1,
		RunID:      "01991ea6-b118-7c93-8d41-6b2f7ae05c19",
		Step:       1,
		RecordKind: "observation",
	}

	for _, one := range []struct {
		name string
		page func(w io.Writer) error
	}{
		{"runs over a Journal that holds entries", func(w io.Writer) error {
			return runsPage(w, []render.Row{entry}, false)
		}},
		{"runs over an empty Journal", func(w io.Writer) error {
			return runsPage(w, nil, false)
		}},
		{"records over a Store that holds versions", func(w io.Writer) error {
			return recordsPage(w, []render.Row{version}, false)
		}},
		{"records over an empty Store", func(w io.Writer) error {
			return recordsPage(w, nil, false)
		}},
	} {
		t.Run(one.name, func(t *testing.T) {
			var page bytes.Buffer
			if err := one.page(&page); err != nil {
				t.Fatal(err)
			}

			written := page.String()
			if !strings.HasPrefix(written, store.Location+"\n\n") {
				t.Errorf("the page begins %q; where the record is stands above the rows, a fact about the answer as a whole being met before them or not at all", written)
			}
			// The three claims, each of them a wrong answer
			// somebody gave: the branch nothing on the surface
			// named, the checkout a reader went looking for, and
			// the portability the session denied out loud.
			for _, claim := range []string{store.BranchName, "never checked out", "travels with a clone"} {
				if !strings.Contains(written, claim) {
					t.Errorf("the page never says %q: %q", claim, written)
				}
			}
		})
	}
}

// TestInspectionListings_NameTheBranchOnce is the other half of the change: the
// empty sentences named the branch themselves, and a page that now carries the
// location above them would say it twice in three lines.
//
// It is a case rather than a comment because the sentence that was shortened is
// the one a reader reaches for when they are editing this page, and saying one
// thing twice is what a surface drifts into rather than what it is written as.
func TestInspectionListings_NameTheBranchOnce(t *testing.T) {
	for _, one := range []struct {
		name string
		page func(w io.Writer) error
	}{
		{"runs", func(w io.Writer) error { return runsPage(w, nil, false) }},
		{"records", func(w io.Writer) error { return recordsPage(w, nil, false) }},
	} {
		t.Run(one.name, func(t *testing.T) {
			var page bytes.Buffer
			if err := one.page(&page); err != nil {
				t.Fatal(err)
			}

			if got := strings.Count(page.String(), store.BranchName); got != 1 {
				t.Errorf("the empty page names %s %d times, want once: %q", store.BranchName, got, page.String())
			}
		})
	}
}
