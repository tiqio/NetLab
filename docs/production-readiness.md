# Production Readiness

NetLab has no application login and therefore requires a trusted management network. Production must
use one authoritative service and database. Validation instances must bind to loopback or an otherwise
isolated namespace and must never be advertised as production.

Before approval, an operator must apply `deploy/nftables/netlab-management.nft`, verify every HTTP,
MCP, console, VNC, capture, and validation listener, and run `scripts/verify-production-authority.sh`.

Credentials exposed during development must be rotated out of band. Evidence records only the UTC
rotation time and reviewer identity; it never records the old or new value. The operator must then set
`credential_rotation_attestation` in `compliance/deployment-authority.json` with
`secret_value_recorded` set to `false`.
