# Compliance CLI Contract

Executable: `netlab-compliance`

The command is read-only with respect to NetLab runtime state.

## `validate`

```text
netlab-compliance validate \
  --ledger compliance/constitution-ledger.json \
  --deployment compliance/deployment-authority.json \
  --templates compliance/template-readiness.json \
  --evidence-dir compliance/evidence \
  --schemas specs/002-constitution-gap-closure/contracts
```

Exit codes:

- `0`: schemas and cross-references are valid.
- `2`: malformed document or schema failure.
- `3`: invalid evidence/finding/exception reference.
- `4`: stale or contradictory verified finding.
- `5`: deployment authority invariant failure.
- `6`: prohibited-content or redaction failure.

## `report`

```text
netlab-compliance report --ledger compliance/constitution-ledger.json --format markdown
```

Produces a deterministic report ordered by constitution principle and finding ID. The report includes
candidate identity, verified/partial/open/blocked/exception counts, stale evidence, deployment
authority, template readiness, and next actions. It never prints secret values or packet payloads.

## `capture-candidate`

```text
netlab-compliance capture-candidate \
  --binary bin/netlabd \
  --contracts specs/001-network-simulator-platform/contracts \
  --output compliance/evidence/candidate.json
```

Records binary and contract digests, release version, build time when available, and non-secret host
facts. The output conforms to `evidence-record.schema.json` after scenario results are attached.

## Cross-Document Invariants

- Every finding ID is unique.
- Every referenced evidence/exception exists.
- `verified` findings reference accepted, non-stale evidence.
- `accepted_exception` findings reference approved, unexpired exceptions.
- At most one externally reachable authoritative deployment exists per host.
- Genuine template validation requires genuine workload and image metadata.
- A passing acceptance run requires cleanup baseline restoration and a passing redaction scan.
