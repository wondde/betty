<script lang="ts">
  // Right-hand pane: edit any field of the selected task. Every change is saved on its own
  // (status/assignee/date on change, text on blur) against the version last loaded.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { nav } from '../lib/nav.svelte'
  import { areaLabel, priorityLabel, dueOf } from '../lib/tasks'

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

  <div class="links">
    {#if ticket.source_url}<button onclick={() => Betty.OpenURL(ticket.source_url)}>Discord 원문 ↗</button>{/if}
    <button onclick={() => Betty.OpenBoard()}>웹 보드 ↗</button>
  </div>
  <p class="stamp">{#if ticket.created_at}만든 날 {fmt(ticket.created_at)} · {/if}마지막 변경 {fmt(ticket.updated_at)}{#if saving} · 저장 중…{/if}</p>
</section>

<style>
  .detail { width: 320px; flex: none; display: flex; flex-direction: column; gap: 10px; padding: 46px 16px 14px; border-left: 0.5px solid var(--line); overflow-y: auto; }
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
  .mini { flex: none; height: 26px; padding: 0 10px; border-radius: 7px; font-size: 12px; font-weight: 700; background: var(--fill); }
  .none { flex: 1; height: 26px; padding: 0 8px; border-radius: 7px; background: var(--fill); font-size: 12.5px; color: var(--fg3); text-align: left; }
  .none:hover { background: var(--fill-hover); color: var(--fg2); }
  .primary { background: var(--accent); color: #fff; }
  .due { flex: none; font-size: 11px; font-weight: 600; }
  .due.overdue, .due.today { color: var(--red); }
  .due.soon { color: var(--orange); }
  .desc-label { font-size: 12px; color: var(--fg3); }
  .desc { min-height: 140px; flex: 1; resize: none; padding: 8px 10px; font-size: 12.5px; line-height: 1.55; user-select: text; -webkit-user-select: text; }
  .links { display: flex; flex-wrap: wrap; gap: 6px; }
  .links button { padding: 4px 10px; border-radius: 6px; background: var(--fill); font-size: 12px; font-weight: 600; }
  .links button:hover { background: var(--fill-hover); }
  .stamp { margin: 0; font-size: 11px; color: var(--fg3); }
</style>
