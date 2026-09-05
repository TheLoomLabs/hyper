`services/` is what we run — one directory per service, and it is the list that
counts.

Watching them is the lookout's job. It is a hosted thing we page off, it holds one
monitor per service, and `docs/lookout-api.md` is what it does and what it answers
back. There is a name declared here for it already, with the credential it needs.
Nothing in this repository knows how to talk to it yet.

Everything in `services/` went behind the firewall on Tuesday. The lookout cannot
reach in to check them any more, so they have to report in to it themselves, and
not one of them has what it needs to do that with. Get each of them what it needs,
through `hyper`.

Then tell me three things. Which of them belongs to which service. Where each one
is on this machine, exactly, so that I can read it myself — I am pasting these
into the deploy environment by hand this afternoon and I am not asking you to do
it for me. And what this repository now holds about each of them, and what it does
not.

Take this as the approval you are told to wait for. Once `hyper check` is clean
and `hyper review` reads the way you meant it, run it — you do not need to come
back to me in between.
