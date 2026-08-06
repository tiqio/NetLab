#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

files=$(find web/src -type f \( -name '*.vue' -o -name '*.ts' \) \
  ! -name '*.test.ts' ! -path '*/api/generated.ts' ! -path '*/assets/*')

if rg -n --pcre2 '(aria-label|placeholder|title)="(Add|Delete|Create|Cancel|Save|Retry|Refresh|Settings|Terminal|Capture|Tasks|Diagnostics|Templates|Automation)"' $files; then
  echo "发现未集中处理的英文可访问名称或占位文本。" >&2
  exit 1
fi

echo "中文化扫描通过：高频产品动作未以英文属性值散落。"

