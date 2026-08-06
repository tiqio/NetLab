#!/usr/bin/env bash
set -euo pipefail

release_json=${1:?release json required}
output=${2:?output required}
candidate=$(jq -er '.candidate_id | select(length>0)' "$release_json")
generated=$(date -u +%Y-%m-%dT%H:%M:%SZ)
jq -n --arg candidate "$candidate" --arg generated "$generated" '
  def item($key; $runtime): {
    template_key:$key, template_version:"built-in", runtime_kind:$runtime,
    status:"declared_not_runtime_validated", genuine_workload:false, image:null,
    bootstrap:{status:"not_tested",evidence_id:null},
    console:{status:"not_tested",evidence_id:null},
    capabilities:{status:"not_tested",evidence_id:null},
    lifecycle:{status:"not_tested",evidence_id:null},
    cleanup:{status:"not_tested",evidence_id:null}, exception_id:null
  };
  {schema_version:"1.0",candidate_id:$candidate,generated_at:$generated,templates:[
    item("fancywan";"qemu"), item("ubuntu-qemu";"qemu"),
    item("fortigate";"qemu"), item("vyos";"qemu"),
    item("ruijie-router";"qemu"), item("ruijie-switch";"qemu"),
    item("busybox-container";"docker"), item("ubuntu-container";"docker"),
    item("nginx-container";"docker")
  ]}' >"$output"
chmod 0644 "$output"
