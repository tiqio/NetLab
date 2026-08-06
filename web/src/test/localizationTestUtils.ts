import { technicalTerms } from "@/locales";

export function visibleEnglishCandidates(root: ParentNode): string[] {
  const allowed = new Set<string>(technicalTerms);
  return (root.textContent ?? "")
    .split(/\s{2,}|\n/)
    .map((value) => value.trim())
    .filter((value) => /[A-Za-z]{3,}/.test(value))
    .filter(
      (value) =>
        ![...allowed].some((term) => value === term || value.includes(term)),
    );
}
