# Target Cleanup and Leak Baseline — 2026-08-12

## Final Durable State

| Resource | Final value |
|----------|------------:|
| Laboratory nodes | 11 running: 6 QEMU, 5 Docker |
| Network objects | 9 |
| Direct links | 3 connected |
| Network attachments | 11 active |
| Network-object links | 8 connected |
| Captures | 0 |
| Traffic workloads | 0 |
| Traffic filters | 0 |

## Host Baseline

| Resource class | Final value |
|----------------|------------:|
| Named namespaces | 7 |
| Host interfaces | 108 |
| Linux bridges | 25 |
| IPv4 routes, all tables | 19 |
| IPv6 routes, all tables | 0 |
| IPv4 rules | 3 |
| IPv6 rules | 2 |
| Listening sockets at final sample | 125 |
| NetLab service processes | 1 |
| NetLab-owned QEMU processes | 6 |
| Running Docker containers | 5 |
| Capture processes | 0 |

- Namespace names are the seven expected owned `n2r-*`, `n2sw-*` and `nlpc-*` resources.
- No namespace matching test, temporary or `netlab-it` naming remained.
- No test runner, temporary acceptance server or `/tmp/netlab-r21`/`r22` capture process remained.
- The host also runs one EVE-NG QEMU outside NetLab ownership. It is excluded from the NetLab baseline and was not touched.

## Database and Rollback

- `/var/lib/netlab/netlab.db` is at schema migration 16 and `PRAGMA integrity_check` returns `ok`.
- Durable tables contain zero captures, zero traffic workloads and zero traffic filters after acceptance cleanup.
- Both R21 and R22 rollback directories contain six expected files.
- `sha256sum -c SHA256SUMS` passed for the prior binary, configuration, readiness file and compressed online SQLite backup in both rollback directories.
