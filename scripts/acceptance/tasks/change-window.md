`firewall/allow` is the allow-list this machine is actually running — one rule to
a line. It is the only thing here anybody edits by hand today.

`requests/pending` is what people have asked to have opened. `requests/withdrawn`
is what they have asked to have closed. Both are the same one-rule-to-a-line
shape.

`control/` is not ours. Change control keeps it, nothing in this repository
writes to it, and there is already a name declared here for that machine that
admits reads and nothing else — it is how anything here touches `control/`. It
stays that way.

There are two things I do by hand, and I want both of them authored through
`hyper`.

Granting. Everything in `requests/pending` goes on the end of `firewall/allow`,
and `requests/pending` comes back empty.

Revoking. Everything in `requests/withdrawn` comes off `firewall/allow`, and
`requests/withdrawn` comes back empty.

Neither of them touches `firewall/allow` unless change control says we may:
`control/window` reads `open`, `control/freeze` is empty, and `control/approver`
names somebody. That is three things today and it will grow, and when it grows I
want to edit it in one place. So there is one copy of it in this repository, and
both of the two above run that copy rather than one of their own. Where it does
not hold, the job stops and `firewall/allow` is left exactly as it was.

Get it clean under `hyper check` and read it back with `hyper review` until it
says what you meant. Then hand me the diff — nothing here gets run.

Then tell me two things, off what you wrote rather than off the disk. What this
repository now says each of the two may touch, and where it says it. And whether
anything you tried first was declined — what it said, and what you changed.
