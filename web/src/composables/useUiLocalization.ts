import { onBeforeUnmount, onMounted } from "vue";
import { exactTranslations } from "@/locales";

const translatedAttributes = ["aria-label", "title", "placeholder"] as const;

function translateText(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return value;
  const translated = exactTranslations[trimmed];
  if (!translated) return value;
  return value.replace(trimmed, translated);
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
  const elements = root instanceof Element ? [root, ...root.querySelectorAll("*")] : [...root.querySelectorAll("*")];
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
          else if (node.nodeType === Node.TEXT_NODE && node.parentElement) localizeElement(node.parentElement);
        }
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });
  });
  onBeforeUnmount(() => observer?.disconnect());
}

