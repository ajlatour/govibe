# govibe

Make a new Go service from [golang-vibe-starter](https://github.com/ajlatour/golang-vibe-starter):
a standard-library HTTP API with its tests, docs, CI, local stack and decision records. govibe
asks what the service is called, its Go module, its first resource, and which of the template's
optional parts it should have, then writes it, ready to run.

```sh
go install github.com/ajlatour/govibe@latest
govibe new
```

```text
The service's name          orders
Its Go module path          github.com/acme/orders
Its first resource          order        (/items becomes /orders, everywhere)
What should it have?        [x] tracing  [x] perf  [x] release  [x] validation  [x] flags
```

Then:

```sh
cd orders
make dev      # the API on :8080, with an in-memory store: nothing else to start
make check    # gofmt, vet and the tests
```

## Without the questions

Every answer is a flag, and `-yes` asks nothing, for scripts and CI:

```sh
govibe new -name orders -module github.com/acme/orders -resource order -without perf,release -yes
govibe new -name billing -module git.example.com/payments/billing -without all -yes billing-svc
```

| Flag | |
| ---- | - |
| `-name` | The service's name: lower-case letters, digits and dashes. It names the images, the compose project and the docs. |
| `-module` | Its Go module path. On github.com, the workflows and code owners follow it. |
| `-resource`, `-resource-plural` | Its first resource: the template's example, `/items`, under this name. Unset, it stays `/items`. |
| `-without` | Optional parts to leave out, comma-separated, or `all`: `govibe features` lists them. |
| `-no-commit` | Leave the result uncommitted. By default it's the repository's first commit. |
| `[dir]` | Where to make it: a new or empty directory. By default, the name. |

`ACCESSIBLE=1 govibe new` asks one question at a time, as plain lines, for a screen reader.

## What it does

govibe carries a snapshot of the template, at a pinned version (`govibe version` says which). It
unpacks it, and runs the template's own `make init` with your answers. That's the same code the
template's CI tests, for every feature left out and the resource renamed, so what govibe makes
builds and passes its tests. Leaving a part out removes it whole: its code, settings, log lines,
docs, tests, CI jobs and containers, and marks its decision records deprecated. govibe needs Go and
git; the service it makes needs nothing else to run (`make dev`), and Docker for its full local
stack.

## Developing govibe

```sh
make check    # gofmt, vet and the unit tests
make e2e      # make services with govibe, and run their tests (minutes)
make sync     # take the template's main into the snapshot (REF=v0.7.0 for a tag)
```

A release of govibe pins the template to a release of its own: `make sync REF=vX.Y.Z`, commit, tag.

## License

[MIT](LICENSE).
