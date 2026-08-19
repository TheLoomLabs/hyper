package cadence

import (
	"math/big"
	"strconv"
	"strings"
)

// The rate is the expression's matches over one full Gregorian cycle, divided
// by 4,800. The cycle is 400 years — 146,097 days, 4,800 months, and exactly
// 20,871 weeks — so it repeats the leap pattern and the weekday alignment
// together, and every one of both appears in it in its true proportion.
//
// That is what makes the denominator derived rather than named (ADR-0066). A
// calendar year has no defensible value: a leap year and a common year
// disagree, and so do two common years starting on different weekdays, so
// `0 3 * * 1` is 4.33 or 4.42 depending on which one somebody picked. Counting
// over *this* year is worse than picking one, because the number then changes
// on 1 January with no edit anywhere. Over the cycle there is nothing to pick
// and no clock to read: `0 3 * * 1` is 20,871 ÷ 4,800 = 4.348, forever and in
// both environments.
//
// The matches are counted rather than reached for in closed form. The
// day-of-month/day-of-week disjunction turns a closed form into a special-case
// table, and the count runs once per rendering (§10).

const (
	// cycleYears is the Gregorian cycle, and cycleMonths is the
	// denominator every rate is stated over — derived from it rather than
	// named beside it.
	cycleYears  = 400
	cycleMonths = cycleYears * 12
	// cycleFirstYear and cycleFirstWeekday anchor the count's weekdays to
	// the calendar the world keeps: 1 January 2000 was a Saturday. Any 400
	// consecutive years hold one whole cycle, so which one is counted
	// changes nothing; what it must not get wrong is which weekday the
	// count opens on.
	cycleFirstYear    = 2000
	cycleFirstWeekday = 6
)

// matches counts what the expression selects over the cycle: the days it
// matches, each carrying one match per minute-and-hour pair the time fields
// select.
//
// The day rule is POSIX's, and it is the one the phrase spends its words on:
// where the day-of-month and day-of-week fields are both restricted, a day
// matching *either* is matched. It reads which fields speak through
// restrictedDayFields, the same derivation the phrase's disjunction reads, so
// the sentence and the number beside it cannot disagree about POSIX.
func (e expr) matches() int64 {
	perDay := int64(len(e[minute].values())) * int64(len(e[hour].values()))

	months, days, weekdays := e[monthOfYear].taken(), e[dayOfMonth].taken(), e[dayOfWeek].taken()
	dayMatches := e.dayRule(days, weekdays)

	var matched int64
	weekday := cycleFirstWeekday
	for year := cycleFirstYear; year < cycleFirstYear+cycleYears; year++ {
		for month := 1; month <= 12; month++ {
			for day, last := 1, daysInMonth(year, month); day <= last; day++ {
				if months[month] && dayMatches(day, weekday) {
					matched++
				}
				weekday = (weekday + 1) % 7
			}
		}
	}
	return matched * perDay
}

// dayRule is the disjunction and the three readings that are not one, decided
// from the fields once and asked of each of the cycle's 146,097 days.
func (e expr) dayRule(days, weekdays []bool) func(day, weekday int) bool {
	dayOfMonthSpeaks, dayOfWeekSpeaks := e.restrictedDayFields()
	switch {
	case dayOfMonthSpeaks && dayOfWeekSpeaks:
		return func(day, weekday int) bool { return days[day] || weekdays[weekday] }
	case dayOfMonthSpeaks:
		return func(day, _ int) bool { return days[day] }
	case dayOfWeekSpeaks:
		return func(_, weekday int) bool { return weekdays[weekday] }
	}
	return func(int, int) bool { return true }
}

// daysInMonth is the Gregorian calendar's own arithmetic, read from the year
// and the month and from no clock.
func daysInMonth(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return 31
}

// renderRate is the count over the cycle rendered as a rate: two significant
// figures, the unit fixed at runs per month on every surface, and the `≈`
// exactly where the number was rounded.
//
// A unit that varied with the value would destroy the comparison the rate
// exists for, and two figures is what the range needs: one decimal place would
// render the rarest expression the grammar admits, `0 0 29 2 *` at 0.0202, as
// `0.0 runs/month` beside a Procedure that does run — a gloss dropping the fact
// it was rendered to carry.
//
// The `≈` is absent where the number is exact, which an exact denominator is
// what makes informative: `0 0 1 * *` is `1 run/month`, exactly, and saying so
// is free. The singular is used at exactly 1 and nowhere else.
func renderRate(matches int64) (text string, value float64) {
	// An expression the calendar has no instance of — the 30th of
	// February — matches nothing, and zero has no leading figure to round
	// to two. It is rated exactly rather than excepted: the grammar admits
	// the expression, so it gets a number like any other, and it is the
	// true one.
	if matches == 0 {
		return "0 runs/month", 0
	}

	exact := big.NewRat(matches, cycleMonths)
	digits, rounded := twoSignificantFigures(exact)
	if rounded.Cmp(exact) == 0 {
		// An exact rate is stated in the digits it has rather than
		// padded out to two: the second figure is a precision claim,
		// and `1.0 runs/month` claims one the arithmetic did not make.
		digits = trimTrailingZeros(digits)
	}

	unit := " runs/month"
	if exact.Cmp(big.NewRat(1, 1)) == 0 {
		unit = " run/month"
	}
	approximately := ""
	if rounded.Cmp(exact) != 0 {
		approximately = "≈"
	}

	// The number the page renders is the number the wire carries, so the
	// float is read back off the rendered digits rather than derived beside
	// them (§10, ADR-0066).
	value, _ = strconv.ParseFloat(digits, 64)
	return approximately + digits + unit, value
}

// twoSignificantFigures rounds a rate to two figures and renders the digits,
// answering what it rendered in both forms so that the caller can ask whether
// anything was lost.
//
// It is exact arithmetic throughout: the `≈` is a claim about whether a
// rounding happened, and a float deciding it would make that claim about its
// own representation error.
func twoSignificantFigures(v *big.Rat) (digits string, rounded *big.Rat) {
	ten := big.NewRat(10, 1)
	one := big.NewRat(1, 1)

	// The exponent: the power of ten the value's leading figure stands in.
	scaled := new(big.Rat).Set(v)
	exponent := 0
	for scaled.Cmp(ten) >= 0 {
		scaled.Quo(scaled, ten)
		exponent++
	}
	for scaled.Cmp(one) < 0 {
		scaled.Mul(scaled, ten)
		exponent--
	}

	// The two figures themselves, rounded half up. Rounding 9.96 up lands
	// on 100, which is the same value one place further left.
	figures := roundHalfUp(scaled.Mul(scaled, ten))
	if figures == 100 {
		figures, exponent = 10, exponent+1
	}

	places := 1 - exponent
	rounded = new(big.Rat).SetInt64(figures)
	for i := 0; i < places; i++ {
		rounded.Quo(rounded, ten)
	}
	for i := 0; i > places; i-- {
		rounded.Mul(rounded, ten)
	}
	return decimal(figures, places), rounded
}

// roundHalfUp is a rational rounded to a whole number, away from zero at the
// half. The rate is never negative, so away from zero is up.
func roundHalfUp(v *big.Rat) int64 {
	whole := new(big.Int).Quo(v.Num(), v.Denom())
	remainder := new(big.Rat).Sub(v, new(big.Rat).SetInt(whole))
	if remainder.Cmp(big.NewRat(1, 2)) >= 0 {
		whole.Add(whole, big.NewInt(1))
	}
	return whole.Int64()
}

// decimal is the two figures written out at the place they stand in: 43 at one
// place is `4.3`, 20 at three is `0.020`, and 88 at none is `8800`.
func decimal(figures int64, places int) string {
	written := strconv.FormatInt(figures, 10)
	if places <= 0 {
		return written + strings.Repeat("0", -places)
	}
	if places >= len(written) {
		return "0." + strings.Repeat("0", places-len(written)) + written
	}
	return written[:len(written)-places] + "." + written[len(written)-places:]
}

// trimTrailingZeros takes the padding off an exact number's fraction, and the
// point with it where nothing is left to the right of it.
func trimTrailingZeros(digits string) string {
	if !strings.Contains(digits, ".") {
		return digits
	}
	return strings.TrimSuffix(strings.TrimRight(digits, "0"), ".")
}
