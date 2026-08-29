`tenants/roster` names the three tenants we have signed. None of them is set up
yet.

Setting one up is two things. There is a shared config every tenant inherits —
`skel/shared.conf` is what it starts out as, and it belongs at
`tenants/shared.conf`. That one gets written once and never again: the moment it is
there we edit it in place, and a second copy over the top of it loses those edits.
Then each name on the roster gets a directory of its own under `tenants/`, with its
own copy of `skel/tenant.conf` in it — and those get edited by hand afterwards too.

Do that through `hyper`, for everyone on the roster.

Then `hooli` signs, the same afternoon. Put it on the roster and set it up the
same way — over the whole roster, not over the one name that is new. I want this
repository able to tell me the other three were considered and left alone rather
than never asked about, and I want none of them touched a second time.

I also want this on a clock. 06:00 every day, so a name somebody adds to the roster
overnight is set up before anyone is awake.

Then tell me two things, and read them off this repository rather than off the
disk. Of the four names, which one did the second run act on, and which did it
leave alone. And what this repository's account says moved between the first run
and the second — all of it, not only the part under `tenants/`.

If some part of what I have asked for is not something `hyper` will do, that is an
answer too. Tell me which part, what it said, and what you did instead.

Take this as the approval you are told to wait for. Once `hyper check` is clean and
`hyper review` reads the way you meant it, run it — you do not need to come back to
me in between.
