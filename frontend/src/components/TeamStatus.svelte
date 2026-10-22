<script lang="ts">
  // Who has how much: open, in progress, blocked, overdue and due soon per person, plus what
  // they finished this week. Click a person to open their list.
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app } from '../lib/view.svelte'
  import { go } from '../lib/nav.svelte'
  import { daysUntil, initial } from '../lib/tasks'

  const v = $derived(app.view)
  const weekAgo = Date.now() - 7 * 86400000

  function stats(tickets: Ticket[]) {
    const open = tickets.filter((t) => t.status !== 'done')
    return {
      open: open.length,
      working: open.filter((t) => t.status === 'in_progress').length,
      blocked: open.filter((t) => t.status === 'blocked').length,
      overdue: open.filter((t) => (daysUntil(t.due_date) ?? 1) < 0).length,
      soon: open.filter((t) => { const d = daysUntil(t.due_date); return d !== null && d >= 0 && d <= 2 }).length,
      doneWeek: tickets.filter((t) => t.status === 'done' && Date.parse(t.updated_at) >= weekAgo).length,
    }
  }

  const rows = $derived.by(() => {
    const all = v.tickets ?? []
    const people = (v.members ?? []).map((m) => ({ id: m.id, name: m.name, me: m.id === v.member?.id, ...stats(all.filter((t) => t.assignee_id === m.id)) }))
    people.sort((a, b) => b.open - a.open || a.name.localeCompare(b.name))
    return people
  })
  const unassigned = $derived(stats((v.tickets ?? []).filter((t) => !t.assignee_id)))
  const max = $derived(Math.max(1, ...rows.map((r) => r.open), unassigned.open))
</script>

<div class="team">
  <div class="legend">
    <span><i class="working"></i>진행 중</span><span><i class="blocked"></i>막힘</span><span><i class="todo"></i>할 일</span>
    <span class="note">막대 길이 = 남은 일</span>
  </div>
  <table>
    <thead>
      <tr><th>팀원</th><th class="bar-col">남은 일</th><th>진행</th><th>막힘</th><th>마감 지남</th><th>2일 안 마감</th><th>이번 주 완료</th></tr>
    </thead>
    <tbody>
      {#each rows as r (r.id)}
        <tr onclick={() => go({ kind: 'member', id: r.id })} title="{r.name}님 업무 보기">
          <td class="who"><span class="avatar">{initial(r.name)}</span>{r.name}{r.me ? ' (나)' : ''}</td>
          <td class="bar-col">
            <div class="bar" style:width="{(r.open / max) * 100}%">
              <i class="working" style:flex={r.working}></i><i class="blocked" style:flex={r.blocked}></i><i class="todo" style:flex={r.open - r.working - r.blocked}></i>
            </div>
            <span class="num">{r.open}</span>
          </td>
          <td>{r.working || ''}</td>
          <td class:warn={r.blocked}>{r.blocked || ''}</td>
          <td class:red={r.overdue}>{r.overdue || ''}</td>
          <td class:orange={r.soon}>{r.soon || ''}</td>
          <td class="green">{r.doneWeek || ''}</td>
        </tr>
      {/each}
      <tr class="unassigned" onclick={() => go({ kind: 'unassigned' })} title="담당자 없는 업무 보기">
        <td class="who"><span class="avatar none">?</span>담당자 없음</td>
        <td class="bar-col"><div class="bar" style:width="{(unassigned.open / max) * 100}%"><i class="todo" style:flex={1}></i></div><span class="num">{unassigned.open}</span></td>
        <td>{unassigned.working || ''}</td>
        <td class:warn={unassigned.blocked}>{unassigned.blocked || ''}</td>
        <td class:red={unassigned.overdue}>{unassigned.overdue || ''}</td>
        <td class:orange={unassigned.soon}>{unassigned.soon || ''}</td>
        <td></td>
      </tr>
    </tbody>
  </table>
</div>

<style>
  .team { padding: 0 18px 20px; }
  .legend { display: flex; gap: 14px; align-items: center; font-size: 11.5px; color: var(--fg2); margin-bottom: 8px; }
  .legend span { display: inline-flex; align-items: center; gap: 5px; }
  .legend i { width: 10px; height: 10px; border-radius: 3px; }
  .note { margin-left: auto; color: var(--fg3); }
  table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
  th { text-align: right; font-size: 11px; font-weight: 600; color: var(--fg3); padding: 6px 8px; border-bottom: 0.5px solid var(--line); white-space: nowrap; }
  th:first-child, th.bar-col { text-align: left; }
  td { padding: 7px 8px; text-align: right; border-bottom: 0.5px solid var(--line); font-variant-numeric: tabular-nums; }
  tr:hover td { background: var(--fill); }
  tbody tr { cursor: default; }
  .who { text-align: left; display: flex; align-items: center; gap: 8px; font-weight: 600; white-space: nowrap; }
  .avatar { width: 22px; height: 22px; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 10.5px; color: var(--fg2); }
  .avatar.none { background: transparent; box-shadow: inset 0 0 0 1px var(--line); }
  .bar-col { width: 40%; text-align: left; }
  .bar-col { position: relative; }
  .bar { display: inline-flex; height: 10px; border-radius: 5px; overflow: hidden; min-width: 2px; vertical-align: middle; max-width: calc(100% - 30px); }
  .num { margin-left: 8px; color: var(--fg2); font-weight: 600; }
  i.working { background: var(--accent); }
  i.blocked { background: var(--red); }
  i.todo { background: color-mix(in srgb, var(--fg3) 55%, transparent); }
  .warn, .red { color: var(--red); font-weight: 700; }
  .orange { color: var(--orange); font-weight: 700; }
  .green { color: var(--green); font-weight: 600; }
  .unassigned .who { color: var(--fg2); }
</style>
