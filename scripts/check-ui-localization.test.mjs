import assert from "node:assert/strict";
import {
  localizationCandidate,
  scanLocalizationSource,
} from "./check-ui-localization.mjs";

assert.equal(localizationCandidate("QEMU · TCP 443"), undefined);
assert.equal(localizationCandidate("running"), "running");
assert.equal(localizationCandidate("Traffic Filter"), "Traffic Filter");
assert.deepEqual(
  scanLocalizationSource(
    '<template><button aria-label="Close session">Reconnect</button></template>',
    "fixture.vue",
  ),
  [
    "fixture.vue: text: Reconnect",
    "fixture.vue: attribute: Close session",
  ],
);
assert.deepEqual(
  scanLocalizationSource(
    '<template><span>Wireshark · TCP 443 · IPv4</span></template>',
    "fixture.vue",
  ),
  [],
);

console.log("中文化扫描规则测试通过。");
