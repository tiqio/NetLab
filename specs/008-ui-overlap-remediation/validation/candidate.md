# Candidate Record

- Candidate ID: `ui-overlap-20260806T085508Z-r2`
- Version: `008-ui-overlap-remediation`
- Source commit: `457e4958065121299c525e07db30771d34e42059`
- Built at: `2026-08-06T09:08:26Z`
- Binary: `bin/netlabd`
- Installed binary SHA-256: `9db2583aaa8b1e46844f25cf0fa36d60dbd19192388d114325e50389f0ec3b7a`
- Embedded binary identity: `sha256:8d608375ce1083867949c360308930e468e5322d8ee1478a0590bb497861a356`
- Contract SHA-256: `sha256:56c691baa26969f1a946d830fd1e8be81347db550a9373e6f4dfff13871b4dfc`
- Schema state: no database migration or SQLite schema change is introduced by feature `008-ui-overlap-remediation`.
- Build source state: clean before candidate build.
- Build command: `VERSION=008-ui-overlap-remediation CANDIDATE_ID=ui-overlap-20260806T085508Z-r2 BINARY_DIGEST=sha256:... CONTRACT_DIGEST=sha256:... BUILT_AT=2026-08-06T09:08:26Z make build`.
- Artifact note: authoritative release metadata and template-readiness candidate identity were synchronized before service restart.

## Overlap Hotfix Candidate

- Candidate ID: `ui-overlap-hotfix-20260806T095807Z-r2`
- Source commit: `d5cfafde226ec351c5daef24ab8ceac84835a971`
- Focused fix commit: `8cc0318803988c3d47cf0373eb04ad8a0e416369`
- Built at: `2026-08-06T09:58:07Z`
- Installed binary SHA-256: `cb2aa0fd2575dedbed0f0b7175a057a01f2726181a6ba9784873d6c859f70969`
- Contract SHA-256: `sha256:005a63ebd6d9b04dfc83a07721b97f2b5f0c1f544be642c8ffa1db7e930fe4a6`
- Source state: clean after the tracked SPA bundle was regenerated and committed.
- Scope: suppress undeclared ECharts legends, move the topology category legend away from the top toolbar and Inspector toggle, and localize the Lightweight/network category labels.

## Theme and Inspector Simplification Candidate

- Candidate ID: `ui-simplify-theme-20260806T101629Z`
- Source commit: `1c31cc93e757fbd6afe7594f80120048d8839f00`
- Focused source commit: `9c35624`
- Built at: `2026-08-06T10:16:29Z`
- Installed binary SHA-256: `343f11cea4b5d30117db2f357db4880ee05200fb6b9efbd32ffada0d7a99ccab`
- Contract SHA-256: `sha256:005a63ebd6d9b04dfc83a07721b97f2b5f0c1f544be642c8ffa1db7e930fe4a6`
- Source state: clean after committing the regenerated embedded SPA.
- Scope: make live xterm sessions follow light/dark theme changes, expose the resolved system theme in the toolbar, remove the obsolete global link card, conditionally show active network-object attachment controls, and remove six unreferenced legacy Vue components.
