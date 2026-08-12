# US2 Dual-Stack Validation — 2026-08-12

## Gates

| Gate | Result |
|------|--------|
| `go test ./internal/domain ./internal/runtime/linuxnet ./internal/app/reconcile ./internal/api/http ./internal/api/mcp ./tests/integration ./tests/recovery -run 'DualStack|Forward|Route' -count=1` | PASS |
| `go test -p 1 ./...` | PASS |
| `go vet ./...` | PASS |
| `git diff --check` | PASS |
| Router revisioned stop/start with attachment and route verification | PASS |
| Four router-local dual-stack paths, 100 probes each | PASS, 400/400 |
| Ten Ubuntu/VyOS/downstream dual-stack paths, 100 probes each | PASS, 1000/1000 |

## Recovery Result

- Missing Docker routes are repaired from desired state after real kernel FIB readback; the ownership file is used only to identify stale NetLab-owned routes.
- IPv4 defaults normalize from kernel `default` to `0.0.0.0/0`.
- IPv6 defaults normalize to `::/0`, and an implicit desired metric accepts the kernel default metric `1024`.
- Restart preserved stable addresses, MACs and attachments while restoring both missing default routes.

## Honest Limitations

- VyOS has no QGA on this image, so guest configuration proof uses the authorized managed serial console.
- No credential, proprietary image content or packet payload is retained.
