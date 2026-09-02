# The lookout API

The lookout watches services and pages us when one stops answering. It holds one
monitor per service. This is what it does over HTTP.

Every path is under `/v1`. Everything that comes back with a body comes back as
JSON, and carries a `request_id` you can quote at us.

## Getting in

Every call carries a bearer token:

```
Authorization: Bearer <token>
```

Without one, or with one we do not know, the call gets `401` and nothing else
happens.

## What a monitor is

```json
{
  "ref": "mon_0d3e88",
  "service": "ingest",
  "window": 60,
  "muted": false,
  "state": "active",
  "created": "2026-01-14T10:02:47Z"
}
```

`ref` is the handle: it is ours to mint, and it is what every call that names one
monitor takes. `service` is what you call the thing being watched. `window` is how
often we look, in seconds. `state` is `pending` on a monitor that has not settled
into a routine with us yet and `active` on one that has. `muted` says whether we
page anyone.

## Listing them

```
GET /v1/monitors
```

```json
{
  "request_id": "req_000002",
  "data": {
    "monitors": [ ... ],
    "cursor": "Mg"
  }
}
```

Oldest first, **two at a time** unless you ask for more: `?limit=` takes anything
from 1 to 100. `data.cursor` is there while there is another page and absent when
there is not — hand it back as `?cursor=` for the next one. A cursor we did not
mint gets `400 invalid_cursor`, and a limit outside the range gets
`400 invalid_limit`.

## Adding one

```
POST /v1/monitors
{"service": "checkout", "window": 60}
```

```json
{
  "request_id": "req_000005",
  "data": {
    "monitor": { ... }
  }
}
```

`201`, and the monitor we made comes back on its own under `data.monitor`.

- `service` is required, and is the name of the thing to watch.
- `window` is required, and is a **whole number of seconds** between 30 and 3600.
  Anything else there gets `400 invalid_window`; a number outside the range gets
  `400 window_out_of_range`.
- `muted` is optional and defaults to `false`.

**One monitor per service.** A service we are already watching gets
`409 already_watched` and nothing is created. Look before you add if you are not
certain what we already hold.

**We look once, straight away.** A service that does not answer that first look
was never watched by us: you still get the `201` and the monitor above, and we do
not keep it. It will not be in the list, and naming its `ref` at either of the
routes below gets `404 no_such_monitor` the way any `ref` we do not hold does. We
do not tell you afterwards, so if it matters to you that we took it on, look.

## The rest of it

```
GET /v1/monitors/{ref}      one monitor, under data.monitor
DELETE /v1/monitors/{ref}   204, and we stop watching it
```

Either of those against a `ref` we do not hold gets `404 no_such_monitor`.

## When we say no

```json
{
  "request_id": "req_000006",
  "error": {
    "code": "already_watched",
    "message": "billing already has a monitor"
  }
}
```

The `code` is the stable half and the `message` is for people.
