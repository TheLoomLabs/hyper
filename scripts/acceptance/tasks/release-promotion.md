`archive/` is immutable, and it is the last word on what a release is. Nothing
in this repository writes to it. There is already a name declared here for that
machine that admits reads and nothing else, and it is how anything here touches
`archive/`. It stays that way.

`live/` is what is serving — the payload, the version it came from, and the
version it was on before that one.

There are two things I do by hand, and I want both of them authored through
`hyper`.

Promoting. `archive/wanted` names the release that ought to be serving. Put it
there, and leave `live/` still able to say what it was on before that.

Rolling back. Put back whatever `live/previous` names, and leave `live/`
saying so.

Neither of them touches `live/` before the archive has been checked: the list of
checksums is there, every payload matches it, the list covers every payload
there is, and the release about to be used is actually in it. That check is four
things today and it will grow, and when it grows I want to edit it in one place.
So there is one copy of it in this repository, and both of the two above run
that copy rather than one of their own.

Get it clean under `hyper check` and read it back with `hyper review` until it
says what you meant. Then hand me the diff — nothing here gets run.

Then tell me two things, off what you wrote rather than off the disk. What this
repository now says each of the two may touch, and where it says it. And whether
anything you tried first was declined — what it said, and what you changed.
