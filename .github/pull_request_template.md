<!-- What changes, and why it is worth changing. The commit messages carry the
     detail; this is the summary a reviewer reads first. -->

## Checks

- [ ] `make check` passes (fmt, vet, tests, build)
- [ ] Golden snapshots reviewed, not just regenerated - an unexpected diff is usually a real regression
- [ ] Keybindings, if any, are defined only in `internal/tui/keys.go`
- [ ] Anything touching Coolify goes through `app.Service`, not straight to HTTP
- [ ] User-visible failures carry an actionable title and suggestion
