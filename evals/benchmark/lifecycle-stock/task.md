Add `Release(item string, qty int) error` to package `stock`. It returns units of an
item that a caller reserved earlier to available stock. Releasing more units of an
item than are currently reserved and not yet released returns the exported error
`ErrOverRelease` and changes nothing. The store is shared by concurrent request
handlers.
