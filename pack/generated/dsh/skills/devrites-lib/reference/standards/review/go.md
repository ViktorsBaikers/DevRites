# Review: Go

> Applies when: reviewing `*.go` files. Load with [`default.md`](default.md).

## Defect probes

- **`err` dropped or shadowed.** `x, _ := f()` on a fallible call without a
  recorded reason; `err` reassigned inside a block so the outer check sees nil;
  `defer f()` discarding an error that matters (e.g. `Close` on a writable
  file, `Rollback` result).
- **Nil-map write.** Assignment into a map declared but never initialized
  (`var m map[K]V; m[k] = v` panics). Reads are safe; writes are not.
- **Goroutine leaks.** A goroutine blocked on a channel send/receive that no
  peer will satisfy after an early return or cancellation; `context.Context`
  created but never passed or cancelled.
- **Timers in loops.** Check effective toolchain, module version, and `GODEBUG`
  settings: under Go 1.23 timer semantics, unreferenced timers are collectible,
  so `time.After` alone does not prove a leak. Older semantics and retained
  references need separate analysis; repeated allocation is a measured cost,
  not automatically a leak. See [timer semantics](https://go.dev/wiki/Go123Timer).
- **`defer` in a loop.** Deferred `Close`/`Unlock` inside a loop accumulates
  until function return — in long-running loops this exhausts handles or holds
  locks far too long.
- **Loop variable capture.** Goroutine or closure capturing the range variable
  (pre-Go-1.22 semantics) — check the module's Go version before flagging.
- **Slice aliasing.** Appending to or mutating a slice derived from a shared
  backing array; `copy` where aliasing was intended or vice versa.
- **`mutex` copy.** Passing a struct containing a `sync.Mutex`/`WaitGroup` by
  value; `go vet` catches some cases — flag the ones it can't (interface
  boxing, slices of lock-bearing structs).
- **Error wrapping broken.** `fmt.Errorf` without `%w` where callers use
  `errors.Is/As`; comparing errors with `==` against a wrapped error.
- **`interface{}` escape.** A concrete invariant (single expected type) hidden
  behind `any` plus a cast that can panic on misuse.
- **HTTP/IO leaks.** `resp.Body` never closed; `io.Reader` partially consumed
  then reused; `json.Decoder` on an untrusted stream without size bound.

## Do not flag

- `if err != nil { return err }` repetition — it is idiomatic, not a defect.
- Missing `context.Context` on a purely synchronous internal helper.
- `//nolint` or `//nosec` that names the suppressed check and gives a reason —
  judge the reason, not the annotation.
- Returning a concrete type instead of an interface — Go convention is
  "accept interfaces, return structs."
- Unexported struct fields in a CLI/internal package — encapsulation pressure
  differs from library code.
