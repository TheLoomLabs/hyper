package run

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
)

// **A `read` Step's Expansion runs concurrently, bounded by the Operation's
// declared limit, and nothing derives from the order its calls complete in**
// (§6, ADR-0045, issue #140).
//
// This is the second exception the milestone's testing note names, and it is
// one for the reason expand_test.go is: what dispatch does is a pure function
// of a member list and a number, and the three facts about it — how many run at
// once, which ones start first, and that the answers come back in Expansion
// order whatever order they arrived in — are not observable from a page. The
// corpus holds the consequence: [testdata/run/a-member-that-reaches-the-deadline]
// drains, and run_concurrency_test.go one package up watches the wire.

// arrivals is what a dispatched call does in place of one: it records that it
// arrived, holds until `width` calls are holding together or the hold elapses,
// and answers the member's name. It is the whole of how the limit is *proved*
// rather than assumed — a run in which nothing ever held together would report
// a peak of one whatever the Manifest declared.
type arrivals struct {
	mutex   sync.Mutex
	inside  int
	peak    int
	order   []string
	waiting chan struct{}
	width   int
	hold    time.Duration
}

// newArrivals is a watcher that lets `width` calls stand together.
func newArrivals(width int, hold time.Duration) *arrivals {
	return &arrivals{waiting: make(chan struct{}), width: width, hold: hold}
}

// enter records one call arriving and holds it there.
func (a *arrivals) enter(name string) {
	a.mutex.Lock()
	a.inside++
	a.peak = max(a.peak, a.inside)
	a.order = append(a.order, name)
	reached := a.inside >= a.width
	release := a.waiting
	if reached {
		// The width is standing together: everything holding is
		// released at once, and the calls after them rendezvous on a
		// channel of their own.
		a.waiting = make(chan struct{})
		close(release)
	}
	a.mutex.Unlock()

	if !reached {
		select {
		case <-release:
		case <-time.After(a.hold):
		}
	}

	a.mutex.Lock()
	a.inside--
	a.mutex.Unlock()
}

// drained takes the whole of a walk, which is what a `read` Step does with one:
// every member attempted, and the two facts about each of them read back in
// Expansion order (§6, step.go).
func drained[T any](limit int, members []member, call func(member) (T, error)) ([]T, []error) {
	concluded := make([]T, len(members))
	faults := make([]error, len(members))
	for taken := range dispatch(limit, members, call) {
		concluded[taken.At], faults[taken.At] = taken.Concluded, taken.Fault
	}
	return concluded, faults
}

// expansionOf is n members named for their position, which is what the dispatch
// order is read off.
func expansionOf(n int) []member {
	members := make([]member, n)
	for i := range members {
		members[i] = member{Name: fmt.Sprintf("member-%02d", i)}
	}
	return members
}

// TestDispatch_TheLimitBoundsWhatIsInFlight is ADR-0045's limit: at most four
// members of one Expansion in flight at once, proved by four standing together
// and no fifth ever joining them.
func TestDispatch_TheLimitBoundsWhatIsInFlight(t *testing.T) {
	watch := newArrivals(4, 2*time.Second)
	drained(4, expansionOf(8), func(m member) (conclusion, error) {
		watch.enter(m.Name)
		return conclusion{name: m.Name}, nil
	})

	if watch.peak != 4 {
		t.Errorf("%d members stood in flight together, want exactly 4 — the limit bounds it and the case proves it is reached", watch.peak)
	}
}

// TestDispatch_ADeclaringManifestIsTheOnlyThingThatBuysConcurrency is the other
// half of the same sentence: a Manifest declaring no `concurrency:` declares 1,
// and its Expansion runs one member at a time.
func TestDispatch_ADeclaringManifestIsTheOnlyThingThatBuysConcurrency(t *testing.T) {
	// The width is two, so a second member arriving while the first is
	// still inside would be seen. None does, and each call pays the hold.
	watch := newArrivals(2, 20*time.Millisecond)
	drained(1, expansionOf(3), func(m member) (conclusion, error) {
		watch.enter(m.Name)
		return conclusion{name: m.Name}, nil
	})

	if watch.peak != 1 {
		t.Errorf("%d members stood in flight together under a limit of 1", watch.peak)
	}
}

// TestDispatch_MembersAreDispatchedInExpansionOrder is §6's *the order above is
// the order members are dispatched in*: under a limit of four, the first four
// members started are the first four of the Expansion — which is what fixes the
// first ten of five hundred.
func TestDispatch_MembersAreDispatchedInExpansionOrder(t *testing.T) {
	watch := newArrivals(4, 2*time.Second)
	drained(4, expansionOf(8), func(m member) (conclusion, error) {
		watch.enter(m.Name)
		return conclusion{name: m.Name}, nil
	})

	first := slices.Clone(watch.order[:4])
	slices.Sort(first)
	want := []string{"member-00", "member-01", "member-02", "member-03"}
	if !slices.Equal(first, want) {
		t.Errorf("the first four members dispatched were %v, want %v", first, want)
	}
}

// TestDispatch_TheAnswersComeBackInExpansionOrder is the rule everything the
// Step writes rests on: **nothing derives from the order calls complete in**.
// The last member answers first here, and the answers are still the Expansion's
// own sequence.
func TestDispatch_TheAnswersComeBackInExpansionOrder(t *testing.T) {
	expansion := expansionOf(4)
	// Each member waits for the one after it to have answered, so the
	// completion order is exactly the reverse of the dispatch order.
	answered := make([]chan struct{}, len(expansion)+1)
	for i := range answered {
		answered[i] = make(chan struct{})
	}
	close(answered[len(expansion)])

	concluded, faults := drained(len(expansion), expansion, func(m member) (conclusion, error) {
		var position int
		fmt.Sscanf(m.Name, "member-%02d", &position)
		<-answered[position+1]
		close(answered[position])
		return conclusion{name: m.Name}, nil
	})

	for position, dispatched := range expansion {
		if concluded[position].name != dispatched.Name {
			t.Errorf("position %d answers %q, want %q", position, concluded[position].name, dispatched.Name)
		}
		if faults[position] != nil {
			t.Errorf("position %d faulted: %v", position, faults[position])
		}
	}
}

// TestDispatch_EveryMemberIsAttempted is the drain: a member that faulted stops
// nothing, and the members after it are called all the same (§6).
func TestDispatch_EveryMemberIsAttempted(t *testing.T) {
	expansion := expansionOf(5)
	var mutex sync.Mutex
	var attempted []string

	concluded, faults := drained(2, expansion, func(m member) (conclusion, error) {
		mutex.Lock()
		attempted = append(attempted, m.Name)
		mutex.Unlock()
		if m.Name == "member-01" {
			return conclusion{}, fmt.Errorf("the world resisted")
		}
		return conclusion{name: m.Name}, nil
	})

	if len(attempted) != len(expansion) {
		t.Errorf("%d members of %d were attempted; every member of a read Expansion is", len(attempted), len(expansion))
	}
	if faults[1] == nil {
		t.Error("the member that faulted answers no fault")
	}
	for position, dispatched := range expansion {
		if position == 1 {
			continue
		}
		if concluded[position].name != dispatched.Name {
			t.Errorf("position %d answers %q, want %q — a member after the fault concluded all the same", position, concluded[position].name, dispatched.Name)
		}
	}
}

// TestDispatch_ALimitBelowOneDispatchesOneAtATime is what a Manifest nobody
// could dispatch under means here: `concurrency: 0` is a number no Step can run
// and this is not the place that judges it, so it runs the Expansion the way
// silence does.
func TestDispatch_ALimitBelowOneDispatchesOneAtATime(t *testing.T) {
	watch := newArrivals(2, 20*time.Millisecond)
	drained(0, expansionOf(2), func(m member) (conclusion, error) {
		watch.enter(m.Name)
		return conclusion{name: m.Name}, nil
	})

	if watch.peak != 1 {
		t.Errorf("%d members stood in flight together under a limit of 0", watch.peak)
	}
}

// TestDispatch_AWalkTheCallerStopsStartsNoMoreMembers is the effectful
// Expansion's stop: a caller that takes no further member calls no further
// member, which is what makes *three of five, then halt* a determinate fact
// rather than a race (§6, step.go).
func TestDispatch_AWalkTheCallerStopsStartsNoMoreMembers(t *testing.T) {
	var mutex sync.Mutex
	var called []string

	for taken := range dispatch(1, expansionOf(5), func(m member) (conclusion, error) {
		mutex.Lock()
		called = append(called, m.Name)
		mutex.Unlock()
		if m.Name == "member-03" {
			return conclusion{}, fmt.Errorf("the world resisted")
		}
		return conclusion{name: m.Name}, nil
	}) {
		if taken.Fault != nil {
			break
		}
	}

	want := []string{"member-00", "member-01", "member-02", "member-03"}
	if !slices.Equal(called, want) {
		t.Errorf("the walk called %v, want %v — the fifth member is never reached", called, want)
	}
}

// TestDispatch_UnderALimitOfOneTheNextMemberWaitsForTheLast is what a serial
// effectful Expansion rests on: the slot is held until the member it belongs to
// has been **taken**, so a version written at the caller's turn is on the branch
// before the next call goes out (§7, step.go).
func TestDispatch_UnderALimitOfOneTheNextMemberWaitsForTheLast(t *testing.T) {
	var mutex sync.Mutex
	var acted []string

	for taken := range dispatch(1, expansionOf(3), func(m member) (conclusion, error) {
		mutex.Lock()
		acted = append(acted, "call "+m.Name)
		mutex.Unlock()
		return conclusion{name: m.Name}, nil
	}) {
		mutex.Lock()
		acted = append(acted, "write "+taken.Concluded.name)
		mutex.Unlock()
	}

	want := []string{
		"call member-00", "write member-00",
		"call member-01", "write member-01",
		"call member-02", "write member-02",
	}
	if !slices.Equal(acted, want) {
		t.Errorf("the walk did %v, want %v — a member is called only once the last has been taken", acted, want)
	}
}
