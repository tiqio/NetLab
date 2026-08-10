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
  ["fixture.vue: text: Reconnect", "fixture.vue: attribute: Close session"],
);
assert.deepEqual(
  scanLocalizationSource(
    `<template><span>{{ active ? "Stop" : "Start" }}</span></template>`,
    "Actions.vue",
  ),
  ["Actions.vue: interpolation: Stop", "Actions.vue: interpolation: Start"],
);
assert.deepEqual(
  scanLocalizationSource(
    "<template><span>Wireshark · TCP 443 · IPv4</span></template>",
    "fixture.vue",
  ),
  [],
);
assert.deepEqual(
  scanLocalizationSource(
    '<template><Dialog :description="dialogDescription" /></template>',
    "PortChooser.vue",
  ),
  [],
);
assert.deepEqual(
  scanLocalizationSource(
    '<script setup>const description = "Delete connection";</script><template />',
    "fixture.vue",
  ),
  ["fixture.vue: runtime: Delete connection"],
);

console.log("中文化扫描规则测试通过。");
