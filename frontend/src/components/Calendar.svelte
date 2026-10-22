<script lang="ts">
  // Month calendar of meetings (Asia/Tokyo dates). Pick a day to see its meetings; add or edit
  // one in the sheet. Saving never sends Discord invites.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Meeting, MeetingInput } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { iso } from '../lib/compose'

  let { onerror, onsaved }: { onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  const today = iso(new Date())
  let month = $state(new Date(new Date().getFullYear(), new Date().getMonth(), 1))
  let day = $state(today)
  let editing = $state<{ id: string; form: MeetingInput; cancelled: boolean } | null>(null)
  let saving = $state(false)
  // An empty WebKit time field shows a placeholder time, so "no time yet" is a button instead.
  let picking = $state<{ start: boolean; end: boolean }>({ start: false, end: false })

  const v = $derived(app.view)
  const meetings = $derived((v.meetings ?? []).slice().sort((a, b) => (a.date + a.start_time).localeCompare(b.date + b.start_time)))
  const byDay = $derived.by(() => {
    const m = new Map<string, Meeting[]>()
    for (const x of meetings) m.set(x.date, [...(m.get(x.date) ?? []), x])
    return m
  })
  // 6 rows × 7 days, weeks start on Monday
  const cells = $derived.by(() => {
    const first = new Date(month)
    const start = new Date(first)
    start.setDate(1 - ((first.getDay() + 6) % 7))
    return Array.from({ length: 42 }, (_, i) => {
      const d = new Date(start)
      d.setDate(start.getDate() + i)
      return { date: iso(d), day: d.getDate(), inMonth: d.getMonth() === first.getMonth(), weekend: d.getDay() === 0 || d.getDay() === 6 }
    })
  })
  const dayMeetings = $derived(byDay.get(day) ?? [])
  const upcoming = $derived(meetings.filter((m) => m.date >= today && m.status !== 'cancelled').slice(0, 6))
  const color = (projectId: string) => v.projects?.find((p) => p.id === projectId)?.color ?? '#8e8e93'
  const projectName = (projectId: string) => v.projects?.find((p) => p.id === projectId)?.name ?? ''
  const label = (d: string) => `${Number(d.slice(5, 7))}월 ${Number(d.slice(8, 10))}일 ${'일월화수목금토'[new Date(d + 'T00:00').getDay()]}요일`
  const when = (m: Meeting) => (m.start_time ? `${m.start_time}${m.end_time ? '–' + m.end_time : ''}` : '시간 미정')

  function shift(n: number) {
    month = new Date(month.getFullYear(), month.getMonth() + n, 1)
  }

  function create() {
    picking = { start: false, end: false }
    editing = { id: '', cancelled: false, form: { project_id: v.projects?.[0]?.id ?? '', title: '', date: day, start_time: '', end_time: '', participants: [], location: '', description: '' } }
  }
  function edit(m: Meeting) {
    picking = { start: false, end: false }
    editing = { id: m.id, cancelled: m.status === 'cancelled', form: { project_id: m.project_id, title: m.title, date: m.date, start_time: m.start_time, end_time: m.end_time, participants: [...(m.participants ?? [])], location: m.location, description: m.description } }
  }
  function toggle(id: string) {
    const f = editing!.form
    f.participants = f.participants?.includes(id) ? f.participants.filter((x) => x !== id) : [...(f.participants ?? []), id]
  }
  async function save() {
    if (!editing) return
    const f = editing.form
    if (!f.title.trim() || !f.date) return onerror('회의 이름과 날짜를 적어 주세요')
    saving = true
    try {
      await Betty.SaveMeeting(editing.id, { ...f, title: f.title.trim(), status: editing.cancelled ? 'cancelled' : 'scheduled' })
      onsaved(editing.id ? '회의를 바꿨어요' : '캘린더에 넣었어요')
      day = f.date
      editing = null
    } catch (e) {
      onerror(message(e))
    } finally {
      saving = false
    }
  }
</script>

<div class="calendar">
  <section class="month">
    <div class="bar">
      <h2>{month.getFullYear()}년 {month.getMonth() + 1}월</h2>
      <div class="nav">
        <button onclick={() => shift(-1)} aria-label="이전 달">‹</button>
        <button onclick={() => ((month = new Date(new Date().getFullYear(), new Date().getMonth(), 1)), (day = today))}>오늘</button>
        <button onclick={() => shift(1)} aria-label="다음 달">›</button>
      </div>
    </div>
    <div class="grid head">{#each ['월', '화', '수', '목', '금', '토', '일'] as w}<span>{w}</span>{/each}</div>
    <div class="grid">
      {#each cells as c (c.date)}
        <button class="cell" class:out={!c.inMonth} class:weekend={c.weekend} class:today={c.date === today} class:picked={c.date === day} onclick={() => (day = c.date)}>
          <span class="n">{c.day}</span>
          {#each (byDay.get(c.date) ?? []).slice(0, 3) as m (m.id)}
            <span class="chip" class:cancelled={m.status === 'cancelled'} style:--c={color(m.project_id)}>{m.start_time ? m.start_time + ' ' : ''}{m.title}</span>
          {/each}
          {#if (byDay.get(c.date)?.length ?? 0) > 3}<span class="more">+{byDay.get(c.date)!.length - 3}</span>{/if}
        </button>
      {/each}
    </div>
  </section>

  <aside class="day">
    <div class="bar">
      <h2>{label(day)}</h2>
      <button class="primary" onclick={create}>＋ 새 회의</button>
    </div>
    {#each dayMeetings as m (m.id)}
      <button class="meeting" class:cancelled={m.status === 'cancelled'} onclick={() => edit(m)}>
        <i style:background={color(m.project_id)}></i>
        <div>
          <b>{m.title}</b>
          <small>{when(m)} · {projectName(m.project_id)}{m.location ? ' · ' + m.location : ''}{m.status === 'cancelled' ? ' · 취소됨' : ''}</small>
        </div>
      </button>
    {:else}
      <p class="none">이날은 회의가 없어요.</p>
    {/each}

    <h3>다가오는 회의</h3>
    {#each upcoming as m (m.id)}
      <button class="meeting small" onclick={() => ((day = m.date), edit(m))}>
        <i style:background={color(m.project_id)}></i>
        <div><b>{m.title}</b><small>{label(m.date)} {m.start_time}</small></div>
      </button>
    {:else}
      <p class="none">잡힌 회의가 없어요.</p>
    {/each}
  </aside>
</div>

<svelte:window onkeydown={(e) => e.key === 'Escape' && editing && (e.stopPropagation(), (editing = null))} />

{#if editing}
  <div class="backdrop" role="presentation" onclick={() => (editing = null)}></div>
  <div class="sheet" role="dialog" aria-modal="true" aria-label={editing.id ? '회의 바꾸기' : '새 회의'}>
    <h1>{editing.id ? '회의 바꾸기' : '새 회의'}</h1>
    <label>회의 이름<input bind:value={editing.form.title} placeholder="예: 다음 업데이트 이야기" /></label>
    <div class="row3">
      <label>날짜<input type="date" bind:value={editing.form.date} /></label>
      {#each [['start', 'start_time', '시작'], ['end', 'end_time', '끝']] as [key, field, name]}
        <label>{name}
          {#if editing.form[field as 'start_time'] || picking[key as 'start']}
            <span class="time">
              <input type="time" bind:value={editing.form[field as 'start_time']} />
              <button type="button" class="clear" aria-label="{name} 시간 지우기" onclick={() => { editing!.form[field as 'start_time'] = ''; picking[key as 'start'] = false }}>✕</button>
            </span>
          {:else}
            <button type="button" class="unset" onclick={() => (picking[key as 'start'] = true)}>시간 미정</button>
          {/if}
        </label>
      {/each}
    </div>
    <label>프로젝트
      <select bind:value={editing.form.project_id} disabled={!!editing.id}>
        {#each v.projects ?? [] as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
      </select>
    </label>
    <fieldset>
      <legend>참석자</legend>
      {#each v.members ?? [] as m (m.id)}
        <button type="button" class="person" class:on={editing.form.participants?.includes(m.id)} onclick={() => toggle(m.id)}>{m.name}</button>
      {/each}
    </fieldset>
    <label>장소<input bind:value={editing.form.location} placeholder="디스코드 음성방, 미팅 주소 등" /></label>
    <label>내용<textarea rows="3" bind:value={editing.form.description}></textarea></label>
    {#if editing.id}<label class="check"><input type="checkbox" bind:checked={editing.cancelled} /> 회의 취소됨으로 표시</label>{/if}
    <p class="note">한국·일본 시간 기준이에요. 저장해도 디스코드 알림이나 초대는 보내지 않아요.</p>
    <div class="actions">
      <button onclick={() => (editing = null)}>취소</button>
      <button class="primary" disabled={saving} onclick={save}>{editing.id ? '저장' : '캘린더에 넣기'}</button>
    </div>
  </div>
{/if}

<style>
  .calendar { height: 100%; display: flex; min-height: 0; }
  .month { flex: 1; min-width: 0; display: flex; flex-direction: column; padding: 0 12px 12px 18px; }
  .bar { display: flex; align-items: center; justify-content: space-between; margin: 4px 0 8px; }
  h2 { margin: 0; font-size: 15px; }
  .nav { display: flex; gap: 4px; }
  .nav button { min-width: 28px; height: 26px; padding: 0 8px; border-radius: 6px; background: var(--fill); font-weight: 600; }
  .grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 2px; }
  .grid:not(.head) { flex: 1; grid-auto-rows: 1fr; }
  .head span { font-size: 11px; color: var(--fg3); text-align: center; padding-bottom: 4px; }
  .cell { display: flex; flex-direction: column; align-items: stretch; gap: 2px; padding: 4px; border-radius: 7px; min-height: 0; overflow: hidden; text-align: left; }
  .cell:hover { background: var(--fill); }
  .cell.picked { background: color-mix(in srgb, var(--accent) 14%, transparent); }
  .cell.out { opacity: 0.4; }
  .cell.weekend .n { color: var(--fg3); }
  .n { font-size: 12px; font-weight: 600; align-self: flex-start; min-width: 20px; height: 20px; line-height: 20px; text-align: center; border-radius: 10px; }
  .cell.today .n { background: var(--red); color: #fff; }
  .chip { font-size: 10.5px; padding: 1px 5px; border-radius: 4px; background: color-mix(in srgb, var(--c) 22%, transparent); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .chip.cancelled { text-decoration: line-through; opacity: 0.6; }
  .more { font-size: 10px; color: var(--fg3); }
  .day { width: 280px; flex: none; padding: 0 16px 12px; border-left: 0.5px solid var(--line); overflow-y: auto; }
  h3 { margin: 18px 0 6px; font-size: 11px; color: var(--fg3); }
  .meeting { display: flex; gap: 10px; width: 100%; padding: 8px; border-radius: 8px; text-align: left; align-items: flex-start; }
  .meeting:hover { background: var(--fill); }
  .meeting i { width: 4px; align-self: stretch; border-radius: 2px; flex: none; }
  .meeting div { display: flex; flex-direction: column; min-width: 0; }
  .meeting b { font-size: 13px; font-weight: 600; }
  .meeting small { font-size: 11.5px; color: var(--fg3); }
  .meeting.cancelled b { text-decoration: line-through; color: var(--fg3); }
  .none { font-size: 12px; color: var(--fg3); padding: 6px 8px; margin: 0; }
  .primary { height: 26px; padding: 0 10px; border-radius: 7px; background: var(--accent); color: #fff; font-weight: 600; font-size: 12px; }
  .backdrop { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.18); z-index: 10; }
  .sheet { position: fixed; left: 50%; top: 52px; transform: translateX(-50%); width: 440px; max-height: calc(100% - 72px); overflow-y: auto; z-index: 11; padding: 18px; border-radius: 14px; background: var(--sheet, #fff); box-shadow: 0 18px 50px rgba(0,0,0,0.25), 0 0 0 0.5px var(--line); display: flex; flex-direction: column; gap: 10px; }
  @media (prefers-color-scheme: dark) { .sheet { --sheet: #2c2c2e; } }
  .sheet h1 { margin: 0; font-size: 17px; }
  .sheet label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--fg2); }
  .sheet input, .sheet select, .sheet textarea { font: inherit; font-size: 13px; color: var(--fg); border: none; outline: none; background: var(--fill); border-radius: 7px; padding: 6px 8px; }
  .sheet input:focus, .sheet textarea:focus, .sheet select:focus { box-shadow: inset 0 0 0 1.5px var(--accent); }
  .time { display: flex; gap: 4px; }
  .time input { flex: 1; min-width: 0; }
  .clear { width: 26px; border-radius: 7px; background: var(--fill); color: var(--fg3); }
  .unset { height: 30px; border-radius: 7px; background: var(--fill); color: var(--fg3); font-size: 12.5px; text-align: left; padding: 0 8px; }
  .unset:hover { color: var(--fg2); }
  .row3 { display: grid; grid-template-columns: 1.4fr 1fr 1fr; gap: 8px; }
  fieldset { border: none; padding: 0; margin: 0; display: flex; flex-wrap: wrap; gap: 5px; }
  legend { font-size: 12px; color: var(--fg2); margin-bottom: 4px; }
  .person { padding: 3px 10px; border-radius: 12px; background: var(--fill); font-size: 12px; }
  .person.on { background: var(--accent); color: #fff; }
  .check { flex-direction: row !important; align-items: center; }
  .note { margin: 0; font-size: 11.5px; color: var(--fg3); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; }
  .actions button { height: 30px; padding: 0 14px; border-radius: 8px; background: var(--fill); font-weight: 600; }
  .actions .primary { background: var(--accent); color: #fff; height: 30px; font-size: 13px; }
</style>
