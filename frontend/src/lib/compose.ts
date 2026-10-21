// ============================================================
// Composer syntax — "@민수 배너 수정 #나루 금요일까지"
// - @name     assignee (exact name or unique prefix)
// - #project  project (spaces ignored, exact or unique prefix)
// - date      오늘 / 내일 / 모레 / (다음주) 월~일요일 / 10/24 / 10월 24일
//             with an optional "~" prefix or "까지" suffix
// Pure functions: no DOM, no network.
// ============================================================

export interface Named { id: string; name: string }
export interface Token { kind: "member" | "project"; query: string; start: number; end: number }
export interface Parsed<M, P> { title: string; assignee: M | null; project: P | null; due: string | null; problems: { kind: "member" | "project"; token: string }[] }

const WEEKDAYS = ["일", "월", "화", "수", "목", "금", "토"];
const EN_WEEKDAYS = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"];

const squash = (s: unknown): string => String(s || "").normalize("NFC").replace(/\s+/g, "").toLowerCase();

export function iso(d: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}
const addDays = (d: Date, n: number): Date => new Date(d.getFullYear(), d.getMonth(), d.getDate() + n);

// One whole word → a date, or null. `nextWeek` shifts weekday words by a week.
function wordDate(word: string, today: Date, nextWeek: boolean): Date | null {
  let w = word.replace(/^~/, "").replace(/까지$/, "");
  if (!w) return null;
  const lower = w.toLowerCase();
  if (w === "오늘" || lower === "today") return today;
  if (w === "내일" || lower === "tomorrow") return addDays(today, 1);
  if (w === "모레") return addDays(today, 2);
  let wd = -1;
  const ko = /^([일월화수목금토])(요일|욜)?$/.exec(w);
  if (ko && (ko[2] || nextWeek)) wd = WEEKDAYS.indexOf(ko[1]); // a bare "월" only counts after "다음주"
  if (EN_WEEKDAYS.includes(lower.slice(0, 3)) && /^[a-z]+$/.test(lower) && lower.length >= 3) wd = EN_WEEKDAYS.indexOf(lower.slice(0, 3));
  if (wd >= 0) {
    // Plain weekday: the coming one (today counts). "다음주" + weekday: that day in the
    // following Monday-to-Sunday week, the way Korean calendars count weeks.
    if (!nextWeek) return addDays(today, (wd - today.getDay() + 7) % 7);
    const nextMonday = addDays(today, 7 - ((today.getDay() + 6) % 7));
    return addDays(nextMonday, (wd + 6) % 7);
  }
  const md = /^(\d{1,2})[/.](\d{1,2})$/.exec(w) || /^(\d{1,2})월(\d{1,2})일$/.exec(w);
  if (md) {
    const m = Number(md[1]) - 1, d = Number(md[2]);
    let date = new Date(today.getFullYear(), m, d);
    if (date.getMonth() !== m || date.getDate() !== d) return null; // 2/30 etc.
    if (date < addDays(today, -31)) date = new Date(today.getFullYear() + 1, m, d); // "1/5" in December = next year
    return date;
  }
  return null;
}

function pick<T extends Named>(list: T[], query: string): { match: T | null; candidates: T[] } {
  const q = squash(query);
  if (!q) return { match: null, candidates: list };
  const exact = list.filter((x) => squash(x.name) === q);
  if (exact.length === 1) return { match: exact[0], candidates: exact };
  const starts = list.filter((x) => squash(x.name).startsWith(q));
  if (starts.length === 1) return { match: starts[0], candidates: starts };
  return { match: null, candidates: starts };
}

// Returns { title, assignee, project, due, problems[] }.
export function parseComposer<M extends Named, P extends Named>(text: string, { members = [] as M[], projects = [] as P[], today = new Date() }: { members?: M[]; projects?: P[]; today?: Date } = {}): Parsed<M, P> {
  today = new Date(today.getFullYear(), today.getMonth(), today.getDate());
  const words = String(text || "").trim().split(/\s+/).filter(Boolean);
  const rest: string[] = [];
  const out: Parsed<M, P> = { title: "", assignee: null, project: null, due: null, problems: [] };
  for (let i = 0; i < words.length; i++) {
    const word = words[i];
    if (word.length > 1 && word[0] === "@") {
      const { match } = pick(members, word.slice(1));
      if (match) out.assignee = match;
      else out.problems.push({ kind: "member", token: word });
      continue;
    }
    if (word.length > 1 && word[0] === "#") {
      const { match } = pick(projects, word.slice(1));
      if (match) out.project = match;
      else out.problems.push({ kind: "project", token: word });
      continue;
    }
    const nextWeek = /^다음\s*주$/.test(word) || word === "다음주";
    if (nextWeek && words[i + 1]) {
      const d = wordDate(words[i + 1], today, true);
      if (d) {
        out.due = iso(d);
        i++;
        continue;
      }
    }
    // "10월 24일" written with a space
    if (/^\d{1,2}월$/.test(word) && /^\d{1,2}일(까지)?$/.test(words[i + 1] || "")) {
      const d = wordDate(word + words[i + 1], today, false);
      if (d) {
        out.due = iso(d);
        i++;
        continue;
      }
    }
    const d = wordDate(word, today, false);
    if (d) {
      out.due = iso(d);
      continue;
    }
    if (word === "까지" && out.due) continue;
    rest.push(word);
  }
  out.title = rest.join(" ");
  return out;
}

// The token being typed at the caret, for @ / # autocomplete.
export function activeToken(text: string, caret: number): Token | null {
  const before = String(text).slice(0, caret);
  const m = /(^|\s)([@#])([^\s@#]*)$/.exec(before);
  if (!m) return null;
  return { kind: m[2] === "@" ? "member" : "project", query: m[3], start: before.length - m[3].length - 1, end: caret };
}

export function suggestions<T extends Named>(token: Token | null, { members = [] as T[], projects = [] as T[] }: { members?: T[]; projects?: T[] } = {}): T[] {
  if (!token) return [];
  const list = token.kind === "member" ? members : projects;
  const q = squash(token.query);
  return list.filter((x) => squash(x.name).startsWith(q) || (q && squash(x.name).includes(q))).slice(0, 6);
}

export function insertSuggestion(text: string, token: Token, item: Named): { text: string; caret: number } {
  const value = (token.kind === "member" ? "@" : "#") + item.name.replace(/\s+/g, "");
  const next = text.slice(0, token.start) + value + " " + text.slice(token.end).replace(/^\s+/, "");
  return { text: next, caret: token.start + value.length + 1 };
}
