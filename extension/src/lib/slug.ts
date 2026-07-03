// Port of the Go side's vault.Slugify so captured inbox notes get the
// same filename stems the CLI would produce: letters lowercased, digits
// kept, everything else collapsed to single dashes, trimmed, max 60
// chars, "note" as the empty fallback.

const LETTER = /\p{L}/u;
const DIGIT = /[0-9]/;

export function slugify(label: string): string {
  let mapped = "";
  for (const r of label) {
    if (LETTER.test(r)) mapped += r.toLowerCase();
    else if (DIGIT.test(r)) mapped += r;
    else mapped += " ";
  }
  let s = mapped.trim().replace(/[^a-z0-9]+/g, "-");
  s = s.replace(/^-+/, "").replace(/-+$/, "");
  if (s === "") s = "note";
  if (s.length > 60) {
    s = s.slice(0, 60).replace(/-+$/, "");
  }
  return s;
}
