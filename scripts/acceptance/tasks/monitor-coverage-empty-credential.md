`services/` is what we run — one directory per service, and it is the list that
counts.

Watching them is the lookout's job. It is a hosted thing we page off, it holds one
monitor per service, and `docs/lookout-api.md` is what it does and what it answers
back. There is a name declared here for it already, with the credential it needs.
Nothing in this repository knows how to talk to it yet.

What is on the lookout was put there by hand over about a year, and it has drifted
from `services/`. Get everything in `services/` watched, on a one-minute window,
through `hyper` — and get there without giving anything a second monitor. The
lookout will not have it, and I would rather not find out what a half-finished run
leaves behind. There is one monitor on there for something we retired last year;
leave it alone, it is mine to clean up.

Then tell me three things, and read all three off this repository rather than off
the lookout. Which of the services were already being watched when you started.
Which monitors this repository is now accountable for, and what it holds about
each one. And what it can still tell me about the ones it only ever looked at.

Take this as the approval you are told to wait for. Once `hyper check` is clean
and `hyper review` reads the way you meant it, run it — you do not need to come
back to me in between. If something stops it before it has put anything on the
lookout, leave it stopped: tell me what stopped it, and what you would need to get
past it.
