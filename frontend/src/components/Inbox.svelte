<script lang="ts">
  // "베티가 찾은 일": tasks and meetings the assistant found in conversations, and design
  // questions. Grouped by project (undecided first, since those need a choice), cards stay
  // short until opened, a change proposal shows exactly what it would change, and several
  // proposals for the same task collapse into the newest one.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Proposal, MeetingProposal } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { areaLabel, priorityLabel, statusLabel } from '../lib/tasks'

  let { onerror, onsaved }: { onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  const v = $derived(app.view)
  let picks = $state<Record<string, string>>({})
  let busy = $state<string | null>(null)
  let open = $state<Record<string, boolean>>({})

  const UNDECIDED = '__undecided'
  const project = (id: string) => v.projects?.find((p) => p.id === id)
  const memberName = (id?: string | null) => v.members?.find((m) => m.id === id)?.name
  const unconfirmed = (p: { payload: { _routing?: { confirmed?: boolean | null } | null } }) => p.payload?._routing?.confirmed === false
  const md = (d?: string | null) => (d ? `${Number(d.slice(5, 7))}/${Number(d.slice(8, 10))}` : '')
  const groupOf = (p: { project_id: string; payload: { _routing?: { confirmed?: boolean | null } | null } }) => (unconfirmed(p) ? UNDECIDED : p.project_id)

  // Newest change proposal per task; older ones for the same task ride along to be skipped together.
  const tasks = $derived.by(() => {
    const newest = new Map<string, Proposal>(), older = new Map<string, Proposal[]>()
    const out: Proposal[] = []
    for (const p of [...(v.proposals ?? [])].sort((a, b) => b.created_at.localeCompare(a.created_at))) {
      const key = p.kind === 'update' && p.ticket_id ? p.ticket_id : ''
      if (!key) { out.push(p); continue }
      if (!newest.has(key)) { newest.set(key, p); out.push(p); older.set(p.id, []) }
      else older.get(newest.get(key)!.id)!.push(p)
    }
    return { list: out, older }
  })

  type Item = { kind: 'task'; p: Proposal } | { kind: 'meeting'; m: MeetingProposal }
  const groups = $derived.by(() => {
    const byKey = new Map<string, Item[]>()
    const add = (key: string, item: Item) => (byKey.has(key) ? byKey.get(key)!.push(item) : byKey.set(key, [item]))
    for (const p of tasks.list) add(groupOf(p), { kind: 'task', p })
    for (const m of v.meetingProposals ?? []) add(groupOf(m), { kind: 'meeting', m })
    const order = [UNDECIDED, ...(v.projects ?? []).map((p) => p.id)]
    return [...byKey.entries()]
      .sort((a, b) => (order.indexOf(a[0]) + 1 || 999) - (order.indexOf(b[0]) + 1 || 999))
      .map(([key, items]) => ({ key, items, name: key === UNDECIDED ? '프로젝트를 골라 주세요' : project(key)?.name ?? '프로젝트', color: key === UNDECIDED ? undefined : project(key)?.color }))
  })

  // What a change proposal would actually change on the current task.
  function changes(p: Proposal): string[] {
    const t = v.tickets?.find((x) => x.id === p.ticket_id)
    if (!t) return []
    const out: string[] = []
    const pl = p.payload
    if (pl.title && pl.title !== t.title) out.push(`제목: ${t.title} → ${pl.title}`)
    if (pl.status && pl.status !== t.status) out.push(`상태: ${statusLabel[t.status]} → ${statusLabel[pl.status] ?? pl.status}`)
    if ((pl.assignee_id ?? null) !== (t.assignee_id ?? null)) out.push(`담당: ${memberName(t.assignee_id) ?? '없음'} → ${memberName(pl.assignee_id) ?? '없음'}`)
    if ((pl.due_date ?? null) !== (t.due_date ?? null)) out.push(`마감: ${md(t.due_date) || '없음'} → ${md(pl.due_date) || '없음'}`)
    if (pl.priority && pl.priority !== (t.priority || 'normal')) out.push(`우선순위: ${priorityLabel[t.priority || 'normal']} → ${priorityLabel[pl.priority] ?? pl.priority}`)
    if (pl.area !== undefined && pl.area !== null && pl.area !== (t.area ?? '')) out.push(`종류: ${areaLabel[t.area ?? '']} → ${areaLabel[pl.area] ?? pl.area}`)
    if (pl.description !== undefined && pl.description.trim() !== (t.description ?? '').trim()) out.push('설명이 바뀌어요')
    return out
  }
  const ticketOf = (p: Proposal) => v.tickets?.find((x) => x.id === p.ticket_id)

  async function run(id: string, fn: () => Promise<void>, note: string) {
    busy = id
    try {
      await fn()
      onsaved(note)
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = null
    }
  }

  function approveTask(p: Proposal) {
    if (unconfirmed(p) && !picks[p.id]) return onerror('어느 프로젝트 업무인지 먼저 골라 주세요')
    const older = tasks.older.get(p.id) ?? []
    run(p.id, async () => {
      await Betty.ApproveProposal(p.id, unconfirmed(p) ? picks[p.id] : '')
      for (const o of older) await Betty.DismissProposal(o.id)
    }, '할 일에 넣었어요')
  }
  function dismissTask(p: Proposal) {
    const older = tasks.older.get(p.id) ?? []
    run(p.id, async () => {
      for (const o of [p, ...older]) await Betty.DismissProposal(o.id)
    }, older.length ? `제안 ${older.length + 1}개를 넘겼어요` : '이번엔 넘어갔어요')
  }
  function approveMeeting(m: MeetingProposal) {
    if (unconfirmed(m) && !picks[m.id]) return onerror('어느 프로젝트 회의인지 먼저 골라 주세요')
    if (!m.payload.date) return onerror('날짜가 정해지지 않은 회의는 캘린더에서 날짜를 정해 주세요')
    run(m.id, () => Betty.ApproveMeeting(m.id, unconfirmed(m) ? picks[m.id] : ''), '캘린더에 넣었어요')
  }
</script>

<div class="inbox">
  {#if !(v.proposals?.length || v.meetingProposals?.length || v.designQuestions?.length)}
    <div class="empty">
      <div class="big">✓</div>
      <p><b>지금은 확인할 일이 없어요</b></p>
      <p>대화에서 판단하기 어려운 일이 생기면 여기에 모아 둘게요.</p>
    </div>
  {/if}

  {#each groups as g (g.key)}
    <h2 style:--pc={g.color}>{#if g.color}<i aria-hidden="true"></i>{/if}{g.name} <span>{g.items.length}</span></h2>
    {#each g.items as item (item.kind === 'task' ? item.p.id : item.m.id)}
      {#if item.kind === 'task'}
        {@const p = item.p}
        {@const older = tasks.older.get(p.id) ?? []}
        {@const t = p.kind === 'update' ? ticketOf(p) : undefined}
        {@const diff = p.kind === 'update' ? changes(p) : []}
        <article class="card" class:expanded={open[p.id]}>
          <div class="head">
            <span class="kind" class:update={p.kind === 'update'}>{p.kind === 'update' ? '바꾸기' : '새 할 일'}</span>
            <h3>{p.payload.title}</h3>
          </div>
          {#if p.kind === 'update'}
            <p class="target">{t ? `BT-${t.number} ${t.title}` : '기존 업무'}{older.length ? ` · 같은 업무 제안 ${older.length + 1}개 중 최신` : ''}</p>
            {#if diff.length}<ul class="diff">{#each diff as d}<li>{d}</li>{/each}</ul>{:else}<p class="meta">바뀌는 항목이 없어요. 넘겨도 돼요.</p>{/if}
          {:else}
            <p class="meta">{memberName(p.payload.assignee_id) ? `${memberName(p.payload.assignee_id)}님이 맡아요` : '맡은 사람 없음'}{p.payload.due_date ? ` · ${md(p.payload.due_date)}까지` : ''}</p>
          {/if}
          {#if open[p.id]}
            {#if p.payload.description}<p class="desc">{p.payload.description}</p>{/if}
            <p class="why"><b>찾은 이유</b> {p.reason}</p>
            {#if p.payload.source_url}<button class="link" onclick={() => Betty.OpenURL(p.payload.source_url)}>대화 보러 가기 ↗</button>{/if}
          {/if}
          <div class="actions">
            {#if unconfirmed(p)}
              <select bind:value={picks[p.id]} aria-label="업무 대상 프로젝트">
                <option value="">프로젝트 선택</option>
                {#each v.projects ?? [] as pr (pr.id)}<option value={pr.id}>{pr.name}</option>{/each}
              </select>
            {/if}
            <button class="primary" disabled={busy === p.id} onclick={() => approveTask(p)}>{p.kind === 'update' ? '바꾸기' : '할 일에 넣기'}</button>
            <button disabled={busy === p.id} onclick={() => dismissTask(p)}>넘어가기{older.length ? ` (${older.length + 1})` : ''}</button>
            <button class="more" onclick={() => (open[p.id] = !open[p.id])}>{open[p.id] ? '접기' : '자세히'}</button>
          </div>
        </article>
      {:else}
        {@const m = item.m}
        <article class="card">
          <div class="head">
            <span class="kind meeting">{m.kind === 'update' ? (m.payload.status === 'cancelled' ? '회의 취소' : '회의 변경') : '새 회의'}</span>
            <h3>{m.payload.title}</h3>
          </div>
          <p class="meta">{m.payload.date ? `${md(m.payload.date)} ${m.payload.start_time || '시간 미정'}` : '날짜 미정'}{m.payload.location ? ` · ${m.payload.location}` : ''}</p>
          {#if open[m.id]}<p class="why"><b>찾은 이유</b> {m.reason}</p>{/if}
          <div class="actions">
            {#if unconfirmed(m)}
              <select bind:value={picks[m.id]} aria-label="회의 프로젝트">
                <option value="">프로젝트 선택</option>
                {#each v.projects ?? [] as pr (pr.id)}<option value={pr.id}>{pr.name}</option>{/each}
              </select>
            {/if}
            <button class="primary" disabled={busy === m.id} onclick={() => approveMeeting(m)}>캘린더에 넣기</button>
            <button disabled={busy === m.id} onclick={() => run(m.id, () => Betty.DismissMeeting(m.id), '이번엔 넘어갔어요')}>넘어가기</button>
            <button class="more" onclick={() => (open[m.id] = !open[m.id])}>{open[m.id] ? '접기' : '자세히'}</button>
            {#if m.payload.source_url}<button class="link" onclick={() => Betty.OpenURL(m.payload.source_url)}>대화 ↗</button>{/if}
          </div>
        </article>
      {/if}
    {/each}
  {/each}

  {#if v.designQuestions?.length}
    <h2>디자인 확인 <span>{v.designQuestions.length}</span></h2>
    {#each v.designQuestions as q (q.id)}
      <article class="card">
        <div class="head"><span class="kind design">디자인</span><h3>{q.question}</h3></div>
        {#if open[q.id]}<p class="why">{q.reason}</p>{/if}
        <div class="actions">
          <button class="primary" onclick={() => Betty.OpenURL(q.url)}>피그마에서 보기 ↗</button>
          <button disabled={busy === q.id} onclick={() => run(q.id, () => Betty.DismissDesignQuestion(q.id), '확인했어요')}>확인했어요</button>
          <button class="more" onclick={() => (open[q.id] = !open[q.id])}>{open[q.id] ? '접기' : '자세히'}</button>
        </div>
      </article>
    {/each}
  {/if}
</div>

<style>
  .inbox { display: flex; flex-direction: column; gap: 8px; padding: 0 16px 20px; }
  h2 { position: sticky; top: 0; z-index: 2; display: flex; align-items: center; gap: 6px; margin: 6px -16px 0; padding: 8px 18px 4px; font-size: 12.5px; font-weight: 700; background: var(--sticky); backdrop-filter: blur(12px); }
  h2 i { width: 9px; height: 9px; border-radius: 50%; background: var(--pc); }
  h2 span { font-weight: 500; color: var(--fg3); }
  .card { padding: 10px 12px; border-radius: 11px; background: var(--card); box-shadow: 0 0 0 0.5px var(--line); }
  .head { display: flex; align-items: baseline; gap: 8px; min-width: 0; }
  .kind { flex: none; padding: 1px 6px; border-radius: 5px; font-size: 10.5px; font-weight: 700; color: var(--green); background: color-mix(in srgb, var(--green) 14%, transparent); }
  .kind.update { color: var(--orange); background: color-mix(in srgb, var(--orange) 14%, transparent); }
  .kind.meeting { color: var(--accent); background: color-mix(in srgb, var(--accent) 14%, transparent); }
  .kind.design { color: var(--fg2); background: var(--fill); }
  h3 { margin: 0; font-size: 13.5px; font-weight: 600; line-height: 1.35; min-width: 0; }
  .card:not(.expanded) h3 { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .target { margin: 4px 0 0; font-size: 11.5px; color: var(--fg3); }
  .diff { margin: 4px 0 0; padding-left: 16px; font-size: 12px; line-height: 1.5; color: var(--fg2); }
  .desc { margin: 6px 0 0; font-size: 12.5px; line-height: 1.5; color: var(--fg2); white-space: pre-wrap; user-select: text; -webkit-user-select: text; }
  .meta { margin: 3px 0 0; font-size: 12px; color: var(--fg2); }
  .why { margin: 6px 0 0; padding: 6px 8px; border-radius: 7px; background: var(--fill); font-size: 12px; line-height: 1.5; color: var(--fg2); user-select: text; -webkit-user-select: text; }
  .why b { color: var(--fg3); font-weight: 600; margin-right: 4px; }
  .actions { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: 8px; }
  .actions button, .actions select { height: 26px; padding: 0 11px; border-radius: 7px; background: var(--fill); font-size: 12px; font-weight: 600; border: none; color: inherit; }
  .actions button:hover { background: var(--fill-hover); }
  .actions .primary { background: var(--accent); color: #fff; }
  .actions .more { margin-left: auto; background: transparent; color: var(--fg3); }
  .link { height: auto !important; margin-top: 4px; background: transparent !important; color: var(--accent) !important; padding: 0 !important; font-size: 12px; font-weight: 600; }
  button:disabled { opacity: 0.5; }
  .empty { text-align: center; padding: 80px 20px; color: var(--fg3); line-height: 1.6; }
  .empty .big { font-size: 34px; color: var(--green); }
  .empty p { margin: 4px 0; }
  .empty b { color: var(--fg); font-size: 14px; }
</style>
