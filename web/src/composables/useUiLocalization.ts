import { onBeforeUnmount, onMounted } from "vue";
import { exactTranslations } from "@/locales";

const translatedAttributes = ["aria-label", "title", "placeholder"] as const;

function translateText(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return value;
  const translated = exactTranslations[trimmed];
  if (translated) return value.replace(trimmed, translated);
  const patterns: Array<[RegExp, (match: RegExpMatchArray) => string]> = [
    [/^Actions for (.+)$/i, (match) => `${match[1]} 的操作`],
    [/^Manage (.+)$/i, (match) => `管理 ${match[1]}`],
    [
      /^Close (.+) console workspace$/i,
      (match) => `关闭 ${match[1]} 终端工作区`,
    ],
    [/^Close (.+)$/i, (match) => `关闭 ${match[1]}`],
    [/^Use (.+)$/i, (match) => `使用 ${match[1]}`],
    [/^Add (.+)$/i, (match) => `添加 ${match[1]}`],
    [/^Remove route (\d+)$/i, (match) => `删除路由 ${match[1]}`],
    [/^Capture quota (\d+) percent$/i, (match) => `抓包配额 ${match[1]}%`],
    [/^Task progress (\d+) percent$/i, (match) => `任务进度 ${match[1]}%`],
  ];
  for (const [pattern, replacement] of patterns) {
    const match = trimmed.match(pattern);
    if (match) return value.replace(trimmed, replacement(match));
  }
  return value;
}

export function localizeElement(root: ParentNode): void {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let node: Node | null;
  while ((node = walker.nextNode())) {
    const parent = node.parentElement;
    if (!parent || parent.closest("pre, code, .xterm, canvas")) continue;
    const translated = translateText(node.textContent ?? "");
    if (translated !== node.textContent) node.textContent = translated;
  }
  const elements =
    root instanceof Element
      ? [root, ...root.querySelectorAll("*")]
      : [...root.querySelectorAll("*")];
  for (const element of elements) {
    for (const attribute of translatedAttributes) {
      const value = element.getAttribute(attribute);
      if (!value) continue;
      const translated = translateText(value);
      if (translated !== value) element.setAttribute(attribute, translated);
    }
  }
}

export function useUiLocalization(): void {
  let observer: MutationObserver | undefined;
  onMounted(() => {
    localizeElement(document.body);
    observer = new MutationObserver((records) => {
      for (const record of records) {
        for (const node of record.addedNodes) {
          if (node instanceof Element) localizeElement(node);
          else if (node.nodeType === Node.TEXT_NODE && node.parentElement)
            localizeElement(node.parentElement);
        }
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });
  });
  onBeforeUnmount(() => observer?.disconnect());
}
