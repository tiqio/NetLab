import { readFile, readdir } from "node:fs/promises";
import { extname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const defaultRoots = [
  "web/src/components",
  "web/src/features",
  "web/src/views",
];
const technical = [
  /\b(?:VLAN Filtering|Guest Agent|Apple Silicon|no shutdown|virtio-net-pci|cloud-init|EVE-NG|QEMU|Docker|VNC|SSH|Telnet|Wireshark|MCP|REST|API|HTTP|HTTPS|TCP|UDP|ICMP|NAT|IPv4|IPv6|DHCP(?:v4|v6)?|SLAAC|DNS|CPU|vCPU|RAM|MAC|QMP|QGA|MTU|VLAN|PVID|STP|CIDR|pcapng?|pcap|tcpdump|JSON|MiB|GiB|VirtIO|VMXNET3|e1000e?|rtl8139|NetLab|VPCS|Windows|Linux|macOS|Intel|CLI|KVM|netns|SPA|lease|shutdown|write|Shift|Enter|Escape|(?:eth|ens|enp|tap|br|pnet)\w*\d*)\b/gi,
  /^(?:eth|ens|enp|tap|br|pnet)\w*\d*$/i,
  /^[a-f0-9:#./_-]+$/i,
  /^#[a-f0-9]{3,8}$/i,
  /^(?:\$\{.+\}|\w+\.\w+(?:\([^)]*\))?)$/,
];
const falsePositive = [
  /^ariaLabel$/,
  /^title$/,
  /^value$/,
  /^Promise$/,
  /^OFFICE$/,
  /^Record$/,
  /^(?:description|helperMessage|imageHint|export|username|password)$/,
  /^(?:active|tcp|udp|icmp)$/,
  /^(?:tcp|udp)(?:\s+(?:src|dst))?\s+port\s+\d+$/i,
  /^(?:icmp|tcp|udp)(?:,\s*(?:tcp|udp)\s+(?:src|dst)?\s+port\s+\d+|,\s*(?:src|dst)\s+(?:host|net)\s+[\da-f.:/]+)*$/i,
  /^\$\{/,
  /^\w+\s*\?$/,
  /(?:===|!==|\?\s*['"`]|\|\||&&)/,
  /^\W*(?:v-if|v-for|:|@|\?|!|\[|\()/,
];

async function files(directory) {
  const output = [];
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) output.push(...(await files(path)));
    else if (extname(path) === ".vue") output.push(path);
  }
  return output;
}

export function localizationCandidate(value) {
  const normalized = value
    .replace(/\$\{[^}]+\}/g, " ")
    .replace(/\s+/g, " ")
    .trim();
  if (/\$\{/.test(value) && /[\u3400-\u9fff]/.test(value)) return undefined;
  if (!/[A-Za-z]{3,}/.test(normalized)) return undefined;
  if (falsePositive.some((pattern) => pattern.test(normalized)))
    return undefined;
  let prose = normalized;
  for (const pattern of technical) prose = prose.replace(pattern, " ");
  if (!/[A-Za-z]{3,}/.test(prose)) return undefined;
  if (/^(?:src|dst)\s+(?:host|net)\s+[\da-f.:/]+$/i.test(normalized))
    return undefined;
  return normalized;
}

function stringLiterals(expression) {
  const values = [];
  for (const match of expression.matchAll(
    /(['"`])((?:\\.|(?!\1)[\s\S])*)\1/g,
  )) {
    values.push(match[2]);
  }
  return values;
}

function isControlLiteral(expression, literal) {
  return (
    /(?:===|!==|\.includes\()/.test(expression) &&
    /^(?:requested|queued|starting|streaming|running|stopping|stopped|failed|cancelled|cancelling|succeeded|active|inactive|connected|pending|auto_restore|remain_stopped|qemu|docker|pc|link|node|network_object(?:_link)?|interface)$/i.test(
      literal,
    )
  );
}

export function scanLocalizationSource(source, path = "component.vue") {
  const findings = [];
  const template = source.match(/<template>([\s\S]*?)<\/template>/)?.[1] || "";
  for (const match of template.matchAll(/>([^<>{}]+)</g)) {
    const value = localizationCandidate(match[1]);
    if (value) findings.push(`${path}: text: ${value}`);
  }
  for (const match of template.matchAll(
    /\s(?:aria-label|placeholder|title|label|description|hint)="([^"]+)"/g,
  )) {
    const value = localizationCandidate(match[1]);
    if (value) findings.push(`${path}: attribute: ${value}`);
  }
  for (const match of template.matchAll(
    /\s:(?:aria-label|placeholder|title|label|description|hint)="([^"]+)"/g,
  )) {
    for (const literal of stringLiterals(match[1])) {
      if (isControlLiteral(match[1], literal)) continue;
      const value = localizationCandidate(literal);
      if (value) findings.push(`${path}: dynamic attribute: ${value}`);
    }
  }
  for (const match of template.matchAll(/\{\{([\s\S]*?)\}\}/g)) {
    for (const literal of stringLiterals(match[1])) {
      if (isControlLiteral(match[1], literal)) continue;
      const value = localizationCandidate(literal);
      if (value) findings.push(`${path}: interpolation: ${value}`);
    }
  }
  for (const match of source.matchAll(
    /(?<![:\w])(?:status\.value|message|description|hint|actionLabel|emptyMessage|helperMessage)\s*(?:=|:)\s*["'`]([^"'`]+)["'`]/g,
  )) {
    const value = localizationCandidate(match[1]);
    if (value) findings.push(`${path}: runtime: ${value}`);
  }
  return findings;
}

export async function scanLocalizationRoots(roots = defaultRoots) {
  const findings = [];
  for (const root of roots) {
    for (const path of await files(root)) {
      const source = await readFile(path, "utf8");
      findings.push(...scanLocalizationSource(source, relative(".", path)));
    }
  }
  return findings;
}

const invokedPath = process.argv[1] ? resolve(process.argv[1]) : "";
if (invokedPath === fileURLToPath(import.meta.url)) {
  const roots = process.env.NETLAB_LOCALIZATION_ROOTS
    ? process.env.NETLAB_LOCALIZATION_ROOTS.split(":").filter(Boolean)
    : defaultRoots;
  const findings = await scanLocalizationRoots(roots);
  if (findings.length) {
    console.error("发现未分类的用户可见英文：\n" + findings.join("\n"));
    process.exit(1);
  }
  console.log("中文化扫描通过：产品文本、属性和运行时消息均已分类。");
}
