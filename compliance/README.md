# Compliance Evidence

This directory contains metadata-only release evidence. It must never contain appliance images,
credentials, bootstrap secrets, private keys, packet payloads, database copies, or raw captures.

Tracked JSON records identify the candidate, contract and scope digests, repeatable procedure,
outcome, cleanup baseline, redaction result, and artifact metadata. Runtime topology state remains in
the authoritative NetLab SQLite database and is not duplicated here.
