Set up a snapshot lifecycle in this repository, over the three log directories
`logs/app`, `logs/web` and `logs/db`.

Two Procedures. The first writes a gzipped tar of each of those directories into
`snapshots/`, and each tarball has to be something `hyper` knows it created, so
that `hyper records` lists them. The second retires them again — and it must not
be able to reach anything `hyper` did not create, so deleting a snapshot it has a
record of is the whole of what it may do.

Both should be clean under `hyper check` and read correctly under `hyper review`
when you hand them back.
