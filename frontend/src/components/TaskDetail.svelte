<script lang="ts">
  import { ask } from '../lib/ask.svelte'
  // Right-hand pane: edit any field of the selected task. Every change is saved on its own
  // (status/assignee/date on change, text on blur) against the version last loaded.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Change, Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { nav } from '../lib/nav.svelte'
  import { areaLabel, priorityLabel, dueOf, stageLabel } from '../lib/tasks'

  let { ticket, onerror, onsaved }: { ticket: Ticket; onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  let title = $state('')
  let description = $state('')
  let saving = $state(false)
  let pickingDue = $state(false)
  let dueInput = $state<HTMLInputElement>()
  let lastId = ''

  // Reload the editable text when another task is selected or the server copy changes.
  $effect(() => {
    if (ticket.id !== lastId || !saving) {
      if (ticket.id !== lastId) pickingDue = false
      title = ticket.title
      description = ticket.description
      lastId = ticket.id
    }
  })

  const me = $derived(app.view.member?.id)
  const auto = $derived(app.view.autoAssignments?.[ticket.id])
  const due = $derived(dueOf(ticket))
  const statuses = [
    ['todo', '할 일'],
    ['in_progress', '진행 중'],
    ['blocked', '막힘'],
    ['done', '완료'],
  ]

  async function save(changes: Record<string, unknown>, note = '') {
    saving = true
    try {
      await Betty.Update(ticket.id, changes)
      if (note) onsaved(note)
    } catch (e) {
      onerror(message(e))
      title = ticket.title
      description = ticket.description
    } finally {
      saving = false
    }
  }

  function saveText() {
    const changes: Record<string, unknown> = {}
    if (title.trim() && title.trim() !== ticket.title) changes.title = title.trim()
    if (description !== ticket.description) changes.description = description
    if (Object.keys(changes).length) save(changes)
  }

  async function claim() {
    try {
      await Betty.Claim(ticket.id)
      onsaved('내가 맡았어요')
    } catch (e) {
      onerror(message(e))
    }
  }

  // What the automations know about this task: confirmed progress, linked work, and their changes.
  const progress = $derived(app.view.progress?.[ticket.id])
  const done = $derived(progress?.finished?.length ?? 0)
  const total = $derived(done + (progress?.remaining?.length ?? 0))

  // Checklist = the task's progress record: people and the automations share it.
  let newItem = $state('')
  let editing = $state<string | null>(null)
  let checking = $state(false)
  async function check(e: { action: string; list: string; index: number; text: string }, after?: () => void) {
    checking = true
    try {
      await Betty.EditChecklist(ticket.id, e)
      after?.()
    } catch (err) {
      onerror(message(err))
    } finally {
      checking = false
    }
  }
  function finishEdit(item: { list: string; index: number; step: { text: string } }, value: string) {
    editing = null
    if (value.trim() && value.trim() !== item.step.text) check({ action: 'edit', list: item.list, index: item.index, text: value.trim() })
  }
  const focus = (el: HTMLElement) => { el.focus() }

  async function trash() {
    if (!(await ask(`BT-${ticket.number} "${ticket.title}"을(를) 휴지통으로 옮길까요?\n휴지통에서 되살릴 수 있어요.`, '휴지통으로', true))) return
    try {
      await Betty.Delete(ticket.id)
      nav.selected = null
      onsaved('휴지통으로 옮겼어요')
    } catch (e) {
      onerror(message(e))
    }
  }
  const githubItems = $derived((app.view.github?.items ?? []).filter((i) => i.ticket_ids?.includes(ticket.id)))
  // Betty's "same task?" guesses for this task, confirmed or rejected right here.
  const guesses = $derived(
    (app.view.github?.suggestions ?? [])
      .filter((s) => s.ticket_id === ticket.id)
      .map((s) => ({ s, i: app.view.github?.items?.find((x) => x.repo_id === s.repo_id && x.number === s.number) }))
      .filter((g) => g.i),
  )
  let guessing = $state(false)
  async function decide(repoID: string, number: number, same: boolean) {
    guessing = true
    try {
      if (same) await Betty.LinkGitHub(ticket.id, repoID, number, false, false)
      else await Betty.RejectGitHub(ticket.id, repoID, number)
      onsaved(same ? '연결했어요' : '다른 업무로 기억할게요')
    } catch (e) {
      onerror(message(e))
    } finally {
      guessing = false
    }
  }
  const designs = $derived(app.view.designLinks?.[ticket.id] ?? [])
  const changeLabel: Record<string, string> = { linked: 'GitHub 연결', progressed: '진행 반영', completed: '완료 처리' }
  let changes = $state<Change[] | null>(null)
  let changesFor = ''
  // Reload when another task is picked or this one changes; keep the old list while refreshing.
  $effect(() => {
    const id = ticket.id, key = id + '@' + ticket.updated_at
    if (key === changesFor) return
    if (!changesFor.startsWith(id + '@')) changes = null
    changesFor = key
    Betty.TaskHistory(id)
      .then((items) => { if (changesFor === key) changes = items ?? [] })
      .catch(() => { if (changesFor === key) changes = [] })
  })

  const fmt = (iso?: string) => (iso ? new Date(iso).toLocaleString('ko-KR', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' }) : '')
</script>

<section class="detail" aria-label="업무 상세">
  <header>
    <span class="num">BT-{ticket.number}</span>
    <button class="close" onclick={() => (nav.selected = null)} aria-label="상세 닫기" title="닫기 (Esc)">✕</button>
  </header>

  <textarea class="title" rows="2" bind:value={title} onblur={saveText} onkeydown={(e) => e.key === 'Enter' && !e.isComposing && (e.preventDefault(), (e.target as HTMLElement).blur())} aria-label="제목"></textarea>

  <div class="segmented" role="radiogroup" aria-label="상태">
    {#each statuses as [value, label]}
      <button role="radio" aria-checked={ticket.status === value} class:on={ticket.status === value} class={value} onclick={() => ticket.status !== value && save({ status: value })}>{label}</button>
    {/each}
  </div>

  <dl>
    <dt>담당</dt>
    <dd>
      <select value={ticket.assignee_id ?? ''} onchange={(e) => save({ assignee_id: (e.target as HTMLSelectElement).value || null })}>
        <option value="">담당자 없음</option>
        {#each app.view.members ?? [] as m (m.id)}<option value={m.id}>{m.name}{m.id === me ? ' (나)' : ''}</option>{/each}
      </select>
      {#if ticket.assignee_id !== me}<button class="mini primary" onclick={claim}>내가 할게</button>{/if}
    </dd>
    {#if auto}
      <dt></dt>
      <dd class="auto" title={auto.reason}>자동 배정 · {auto.reason}. 다른 사람 일이면 위에서 바꿔 주세요.</dd>
    {/if}

    <dt>마감</dt>
    <dd>
      <!-- An empty WebKit date field shows today's date, so "no deadline" gets its own button. -->
      {#if ticket.due_date || pickingDue}
        <input bind:this={dueInput} type="date" value={ticket.due_date ?? ''} onchange={(e) => { pickingDue = false; save({ due_date: (e.target as HTMLInputElement).value || null }) }} />
        {#if due}<span class="due {due.tone}">{due.label}</span>{/if}
        {#if ticket.due_date}<button class="mini" onclick={() => save({ due_date: null })} title="마감 지우기" aria-label="마감 지우기">✕</button>{/if}
      {:else}
        <button class="none" onclick={() => { pickingDue = true; queueMicrotask(() => dueInput?.focus()) }}>마감 없음 · 정하기</button>
      {/if}
    </dd>

    <dt>우선순위</dt>
    <dd>
      <select value={ticket.priority || 'normal'} onchange={(e) => save({ priority: (e.target as HTMLSelectElement).value })}>
        {#each Object.entries(priorityLabel) as [value, label]}<option {value}>{label}</option>{/each}
      </select>
    </dd>

    <dt>프로젝트</dt>
    <dd>
      <select value={ticket.project?.id ?? ''} onchange={(e) => save({ project_id: (e.target as HTMLSelectElement).value }, '프로젝트를 옮겼어요')}>
        {#each app.view.projects ?? [] as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
      </select>
    </dd>

    <dt>종류</dt>
    <dd>
      <select value={ticket.area ?? ''} onchange={(e) => save({ area: (e.target as HTMLSelectElement).value })}>
        {#each Object.entries(areaLabel) as [value, label]}<option {value}>{label}</option>{/each}
      </select>
    </dd>
  </dl>

  <label class="desc-label" for="desc">설명</label>
  <textarea id="desc" class="desc" bind:value={description} onblur={saveText} placeholder="설명을 적어 주세요"></textarea>

  <section class="block" aria-label="체크리스트">
    <h4>체크리스트{#if total} <span class="count">{done}/{total}</span>{/if}</h4>
    {#if ticket.status === 'done' && progress?.remaining?.length}<p class="note">완료로 표시된 일이에요. 남은 항목은 완료 전에 정리된 내용이에요.</p>{/if}
    <ul class="steps">
      {#each [...(progress?.remaining ?? []).map((step, index) => ({ step, index, list: 'remaining' })), ...(progress?.finished ?? []).map((step, index) => ({ step, index, list: 'finished' }))] as item (item.list + item.index + item.step.text)}
        <li class:ok={item.list === 'finished'}>
          <button class="box" class:on={item.list === 'finished'} disabled={checking} onclick={() => check({ action: 'toggle', list: item.list, index: item.index, text: '' })} aria-label={item.list === 'finished' ? '체크 해제' : '완료 체크'}>
            <svg viewBox="0 0 14 14" aria-hidden="true"><rect x="1" y="1" width="12" height="12" rx="3.5" /><path d="M4 7.2l2 2L10 5" /></svg>
          </button>
          {#if editing === item.list + item.index}
            <input class="edit" value={item.step.text} onblur={(e) => finishEdit(item, (e.target as HTMLInputElement).value)}
              onkeydown={(e) => { if (e.key === 'Enter' && !e.isComposing) (e.target as HTMLInputElement).blur(); else if (e.key === 'Escape') editing = null }} use:focus />
          {:else}
            <span class="text" role="button" tabindex="-1" ondblclick={() => !item.step.quote && !item.step.url && (editing = item.list + item.index)}
              onkeydown={(e) => e.key === 'Enter' && !item.step.quote && !item.step.url && (editing = item.list + item.index)} title={item.step.quote || item.step.url ? '자동 정리가 남긴 항목이에요' : '두 번 눌러 고치기'}>{item.step.text}</span>
          {/if}
          {#if item.step.url}<button class="evidence" onclick={() => Betty.OpenURL(item.step.url)} title={item.step.quote}>근거</button>{/if}
          <button class="remove" disabled={checking} onclick={() => check({ action: 'remove', list: item.list, index: item.index, text: '' })} aria-label="항목 지우기" title="지우기">✕</button>
        </li>
      {/each}
    </ul>
    <input class="add" bind:value={newItem} placeholder="＋ 항목 추가" disabled={checking}
      onkeydown={(e) => e.key === 'Enter' && !e.isComposing && newItem.trim() && check({ action: 'add', list: 'remaining', index: 0, text: newItem.trim() }, () => (newItem = ''))} aria-label="체크리스트 항목 추가" />
  </section>

  <section class="block" aria-label="연결">
    <h4>연결</h4>
    {#each githubItems as i (i.repo_id + i.number)}
      {@const [label, tone] = stageLabel(i)}
      <button class="link" onclick={() => Betty.OpenURL(i.url)} title={i.title}>
        <span class="kind">{i.kind === 'pr' ? 'PR' : '이슈'} #{i.number}</span><span class="name">{i.title}</span><span class="stage {tone}">{label}</span>
      </button>
    {/each}
    {#each guesses as g (g.s.repo_id + g.s.number)}
      <div class="guess" title={g.s.reason}>
        <button class="name" onclick={() => Betty.OpenURL(g.i!.url)}><span class="q">같은 업무?</span> {g.i!.kind === 'pr' ? 'PR' : '이슈'} #{g.i!.number} {g.i!.title}</button>
        <button class="yes" disabled={guessing} onclick={() => decide(g.s.repo_id, g.s.number, true)} title="같은 업무예요" aria-label="같은 업무예요">✓</button>
        <button class="no" disabled={guessing} onclick={() => decide(g.s.repo_id, g.s.number, false)} title="다른 업무예요" aria-label="다른 업무예요">✕</button>
      </div>
    {/each}
    {#each designs as d (d.url)}
      <button class="link" onclick={() => Betty.OpenURL(d.url)} title={d.reason}><span class="kind">피그마</span><span class="name">{d.label}</span></button>
    {/each}
    {#if ticket.source_url}<button class="link" onclick={() => Betty.OpenURL(ticket.source_url)}><span class="kind">디스코드</span><span class="name">원문 대화</span></button>{/if}
    {#if !githubItems.length && !guesses.length && !designs.length && !ticket.source_url}<p class="note">연결된 PR·피그마·대화가 없어요.</p>{/if}
    <button class="link quiet" onclick={() => Betty.OpenBoard()}><span class="name">웹 보드에서 보기 ↗</span></button>
  </section>

  <section class="block" aria-label="베티가 한 일">
    <h4>베티가 한 일</h4>
    {#if changes === null}
      <p class="note">불러오는 중…</p>
    {:else if !changes.length}
      <p class="note">베티가 자동으로 바꾼 적이 없어요.</p>
    {:else}
      <ul class="changes">
        {#each changes as c (c.id)}
          <li class:undone={c.undone_at}>
            <span class="when">{fmt(c.created_at)} · {changeLabel[c.kind] ?? c.kind}{c.undone_at ? ' · 되돌림' : ''}</span>
            <span class="why">{c.reason}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
  <button class="trash" onclick={trash}>휴지통으로 옮기기</button>
  <p class="stamp">{#if ticket.created_at}만든 날 {fmt(ticket.created_at)} · {/if}마지막 변경 {fmt(ticket.updated_at)}{#if saving} · 저장 중…{/if}</p>
</section>

<style>
  .detail { width: 320px; flex: none; display: flex; flex-direction: column; gap: 10px; padding: var(--top) 16px 14px; border-left: 0.5px solid var(--line); overflow-y: auto; }
  .detail > * { flex-shrink: 0; } /* the pane scrolls; nothing squeezes */
  header { display: flex; align-items: center; justify-content: space-between; }
  .num { font-size: 12px; font-weight: 700; color: var(--fg3); font-variant-numeric: tabular-nums; }
  .close { width: 22px; height: 22px; border-radius: 6px; color: var(--fg3); }
  .close:hover { background: var(--fill-hover); }
  textarea, input, select { font: inherit; color: inherit; border: none; outline: none; background: var(--fill); border-radius: 7px; }
  textarea:focus, input:focus, select:focus { box-shadow: inset 0 0 0 1.5px var(--accent); }
  .title { resize: none; padding: 6px 8px; font-size: 16px; font-weight: 700; line-height: 1.35; background: transparent; }
  .title:hover { background: var(--fill); }
  .segmented { display: grid; grid-template-columns: repeat(4, 1fr); gap: 2px; padding: 2px; border-radius: 8px; background: var(--fill); }
  .segmented button { height: 26px; border-radius: 6px; font-size: 12px; font-weight: 600; color: var(--fg2); }
  .segmented button.on { background: var(--card-solid, #fff); color: var(--fg); box-shadow: 0 1px 2px rgba(0,0,0,0.12); }
  .segmented button.on.in_progress { color: var(--accent); }
  .segmented button.on.blocked { color: var(--red); }
  .segmented button.on.done { color: var(--green); }
  @media (prefers-color-scheme: dark) { .segmented button.on { --card-solid: #48484a; } }
  dl { display: grid; grid-template-columns: 64px 1fr; gap: 8px 10px; align-items: center; margin: 4px 0; }
  dt { font-size: 12px; color: var(--fg3); }
  dd { margin: 0; display: flex; align-items: center; gap: 6px; min-width: 0; }
  select, input[type='date'] { height: 26px; padding: 0 8px; font-size: 12.5px; min-width: 0; flex: 1; }
  .auto { font-size: 11.5px; color: var(--fg3); line-height: 1.4; margin-top: -4px; }
  .mini { flex: none; height: 26px; padding: 0 10px; border-radius: 7px; font-size: 12px; font-weight: 700; background: var(--fill); }
  .none { flex: 1; height: 26px; padding: 0 8px; border-radius: 7px; background: var(--fill); font-size: 12.5px; color: var(--fg3); text-align: left; }
  .none:hover { background: var(--fill-hover); color: var(--fg2); }
  .primary { background: var(--accent); color: #fff; }
  .due { flex: none; font-size: 11px; font-weight: 600; }
  .due.overdue, .due.today { color: var(--red); }
  .due.soon { color: var(--orange); }
  .desc-label { font-size: 12px; color: var(--fg3); }
  .desc { min-height: 120px; flex: none; resize: vertical; padding: 8px 10px; font-size: 12.5px; line-height: 1.55; user-select: text; -webkit-user-select: text; }
  .block { display: flex; flex-direction: column; gap: 4px; }
  h4 { margin: 4px 0 2px; font-size: 12px; font-weight: 600; color: var(--fg3); }
  .note { margin: 0; font-size: 12px; color: var(--fg3); }
  .steps, .changes { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
  .steps li { display: flex; align-items: flex-start; gap: 7px; padding: 2px 0; font-size: 12.5px; line-height: 1.45; }
  .box { flex: none; width: 16px; height: 18px; display: grid; place-items: center; }
  .box svg { width: 14px; height: 14px; fill: none; stroke: var(--fg3); stroke-width: 1.3; }
  .box path { stroke: transparent; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
  .box.on rect { fill: var(--green); stroke: var(--green); }
  .box.on path { stroke: #fff; }
  .box:hover rect { stroke: var(--accent); }
  .ok .text { color: var(--fg3); text-decoration: line-through; }
  .count { font-weight: 500; margin-left: 4px; }
  .remove { flex: none; width: 18px; height: 18px; border-radius: 5px; font-size: 10px; color: var(--fg3); opacity: 0; }
  .steps li:hover .remove { opacity: 1; }
  .remove:hover { background: var(--fill-hover); color: var(--red); }
  .edit, .add { height: 24px; padding: 0 6px; font-size: 12.5px; }
  .edit { flex: 1; min-width: 0; }
  .add { margin-top: 2px; background: transparent; }
  .add:hover, .add:focus { background: var(--fill); }
  .trash { align-self: flex-start; padding: 4px 8px; border-radius: 6px; font-size: 12px; font-weight: 600; color: var(--red); }
  .trash:hover { background: color-mix(in srgb, var(--red) 12%, transparent); }
  .text { flex: 1; min-width: 0; }
  .evidence { flex: none; font-size: 11px; color: var(--accent); }
  .link { display: flex; align-items: center; gap: 6px; min-height: 28px; padding: 4px 8px; border-radius: 7px; background: var(--fill); text-align: left; font-size: 12px; }
  .link:hover { background: var(--fill-hover); }
  .link.quiet { background: none; color: var(--fg3); }
  .guess { display: flex; align-items: center; gap: 4px; min-height: 26px; padding: 2px 4px 2px 8px; border-radius: 7px; border: 1px dashed var(--line); font-size: 11.5px; }
  .guess .name { flex: 1; min-width: 0; text-align: left; color: var(--fg2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .guess .q { font-weight: 700; color: var(--orange); }
  .guess .yes, .guess .no { flex: none; width: 22px; height: 22px; border-radius: 6px; font-size: 11px; }
  .guess .yes { color: var(--green); }
  .guess .no { color: var(--fg3); }
  .guess .yes:hover, .guess .no:hover { background: var(--fill-hover); }
  .kind { flex: none; font-weight: 700; color: var(--fg2); }
  .name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .stage { flex: none; padding: 0 6px; border-radius: 5px; font-size: 10.5px; font-weight: 700; background: var(--card-solid, rgba(255,255,255,0.6)); }
  .stage.merged { color: #8250df; }
  .stage.open { color: var(--green); }
  .stage.review { color: var(--orange); }
  .stage.closed, .stage.draft { color: var(--fg3); }
  .changes li { display: flex; flex-direction: column; gap: 1px; padding: 6px 8px; border-radius: 7px; background: var(--fill); }
  .changes li.undone { opacity: 0.6; }
  .when { font-size: 11px; font-weight: 600; color: var(--fg3); }
  .why { font-size: 12px; line-height: 1.45; color: var(--fg2); display: -webkit-box; -webkit-line-clamp: 3; line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; user-select: text; -webkit-user-select: text; }
  .stamp { margin: 0; font-size: 11px; color: var(--fg3); }
</style>
