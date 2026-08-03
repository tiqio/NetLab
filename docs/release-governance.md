# Release Governance

Every release candidate has one immutable candidate ID and one aggregate acceptance record at
`compliance/evidence/current-candidate.json`. The reviewer records identity and approval outside any
secret-bearing system, then confirms the candidate, binary, contract, and approved-scope digests.

Exceptions require an owner, affected finding, risk statement, approval identity, approval time, and
expiration. Expired or unapproved exceptions block release. Genuine device support is never inferred
from substitute media.

When Git metadata is unavailable, use `make capture-candidate CANDIDATE_ID=...` and retain the binary
and contract digests as the release identity. This fallback does not waive review or acceptance gates.

Run `CANDIDATE_ID=... make constitution-acceptance`. Skipped privileged, browser, host-restart, or
operator-image gates must include a repeatable command and leave the conclusion `blocked` until rerun
on the authoritative host. A release is approved only when the ledger and acceptance report both have
the same candidate and a single ready/passed conclusion.
