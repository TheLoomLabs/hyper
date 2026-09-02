`services/` is what we run — one directory per service, and it is the list that
counts.

Watching them is the lookout's job. It is a hosted thing we page off, it holds one
monitor per service, and `docs/lookout-api.md` is what it does and what it answers
back. There is a name declared here for it already, with the credential it needs.
Nothing in this repository knows how to talk to it yet.

Two things today, in this order.

What is on the lookout was put there by hand over about a year, and it has drifted
from `services/`. Get everything in `services/` watched, on a one-minute window,
through `hyper` — and get there without giving anything a second monitor. The
lookout will not have it, and I would rather not find out what a half-finished run
leaves behind.

Then `pricing` and `warehouse` come off the fleet this afternoon — `pricing` went
at lunchtime, `warehouse` goes after the last batch — so their monitors come off
with them. Theirs and nothing else's. There is older stuff up there that predates
all of this, and where this repository has no account of having put a monitor
somewhere, it is not ours to take off, however much it looks like it belongs to
nobody.

And before you take one off, make sure the lookout still says it is watching what
you think it is.

Then tell me three things, and read all three off this repository rather than off
the lookout. What it is now accountable for up there, and what it holds about each
one. What it took off. And whether anything up there moved that we did not move.

Take this as the approval you are told to wait for. Once `hyper check` is clean
and `hyper review` reads the way you meant it, run it — you do not need to come
back to me in between.
