<script lang="ts">
  // 오늘: the home screen. What I should do now sits on top with its buttons in view, then
  // what waits for my answer, then the rest of my quick work (folded), then long-term work on
  // its own, then my team in a line per person and the other teams as one summary line each.
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, asAdmin } from '../lib/view.svelte'
  import { go, areaName, memberArea, type Area } from '../lib/nav.svelte'
  import { daysUntil, sortTasks } from '../lib/tasks'
  import TaskCard from './TaskCard.svelte'
  import NowDoing from './NowDoing.svelte'

  let { selected, isNew, onselect, onerror }: { selected: string | null; isNew: Set<string>; onselect: (t: Ticket) => void; onerror: (m: string) => void } = $props()

  const v = $derived(app.view)
  const me = $derived(v.member?.id ?? '')
  const all = $derived(v.tickets ?? [])
  const mine = $derived(sortTasks(all.filter((t) => t.assignee_id === me && t.status !== 'done'), isNew, v.order ?? []))
  const urgent = (t: Ticket) => (daysUntil(t.due_date) ?? 99) <= 1
  // Long-term work (Betty's guess or someone's choice) moves slowly on purpose: it gets its own
  // section and stays out of 지금 할 일 unless its deadline is close.
  const isLong = (t: Ticket) => Boolean(v.longTerm?.[t.id])
  // 지금 할 일: due by tomorrow, already started, stuck, or just handed to me.
  const now = $derived(mine.filter((t) => urgent(t) || (!isLong(t) && (t.status !== 'todo' || isNew.has(t.id)))))
  const later = $derived(mine.filter((t) => !now.includes(t) && !isLong(t)))
  const longTerm = $derived(mine.filter((t) => !now.includes(t) && isLong(t)))
  let showLater = $state(false)
  const LATER = 5

  const today = new Date().toDateString()
  const doneToday = $derived(all.filter((t) => t.assignee_id === me && t.status === 'done' && new Date(t.updated_at).toDateString() === today).length)

  // Waiting on me: Betty's finds that would land on me, then the rest of the inbox.
  const finds = $derived((v.proposals?.length ?? 0) + (v.meetingProposals?.length ?? 0) + (v.designQuestions?.length ?? 0))
  const findsForMe = $derived((v.proposals ?? []).filter((p) => p.payload?.assignee_id === me).length)
  const autoMine = $derived(mine.filter((t) => v.autoAssignments?.[t.id]?.member_id === me))

  // My team in detail, the others in one line each.
  const myArea = $derived(memberArea(v.member ?? {}))
  const areaOf = (id?: string | null) => memberArea(v.members?.find((m) => m.id === id) ?? {})
  const teamTasks = $derived(all.filter((t) => t.status !== 'done' && t.assignee_id && t.assignee_id !== me && (myArea === 'none' || areaOf(t.assignee_id) === myArea)))
  const others = $derived.by(() => {
    if (myArea === 'none') return []
    return (['engineering', 'design', 'marketing'] as Area[])
      .filter((a) => a !== myArea)
      .map((a) => {
        const people = (v.members ?? []).filter((m) => memberArea(m) === a)
        const open = all.filter((t) => t.status !== 'done' && t.assignee_id && people.some((m) => m.id === t.assignee_id))
        return { id: a, people: people.length, open: open.length, working: open.filter((t) => t.status === 'in_progress').length, late: open.filter((t) => (daysUntil(t.due_date) ?? 0) < 0).length }
      })
      .filter((a) => a.people)
  })

  const late = $derived(mine.filter((t) => (daysUntil(t.due_date) ?? 0) < 0).length)
  const quickCount = $derived(mine.length - longTerm.length)
  const headline = $derived(
    !mine.length ? '오늘 남은 일이 없어요' : !quickCount ? `길게 보는 일 ${longTerm.length}개, 급한 건 없어요` : now.length ? `내 일 ${quickCount}개` : `내 일 ${quickCount}개, 급한 건 없어요`,
  )
  // Finds are everyone's; they ask for me only when one would land on me (admins sort the rest).
  const showFinds = $derived(finds > 0 && (findsForMe > 0 || asAdmin()))
</script>

<div class="today">
  <p class="headline">{headline}{#if late}<span class="late">마감 지난 일 {late}개</span>{/if}{#if doneToday}<span class="done">오늘 {doneToday}개 끝냈어요</span>{/if}</p>

  {#if now.length}
    <section>
      <h2>지금 할 일</h2>
      {#each now as t (t.id)}
        <div data-task={t.id}><TaskCard ticket={t} isNew={isNew.has(t.id)} selected={selected === t.id} actions onselect={() => onselect(t)} {onerror} /></div>
      {/each}
    </section>
  {/if}

  {#if showFinds || autoMine.length}
    <section>
      <h2>확인해 주세요</h2>
      {#if showFinds}
        <button class="ask" onclick={() => go({ kind: 'inbox' })}>
          <span class="dot">✦</span>
          <span class="text"><b>{findsForMe ? `베티가 대화에서 찾은 내 일 ${findsForMe}개` : `베티가 대화에서 찾은 일 ${finds}개`}</b><small>맞으면 넣고, 아니면 넘기면 돼요</small></span>
          <span class="go">하나씩 보기 ›</span>
        </button>
      {/if}
      {#if autoMine.length}
        <button class="ask" onclick={() => onselect(autoMine[0])}>
          <span class="dot">↺</span>
          <span class="text"><b>베티가 역할을 보고 맡긴 일 {autoMine.length}개</b><small>내 일이 아니면 상세에서 다른 사람에게 넘겨 주세요</small></span>
          <span class="go">보기 ›</span>
        </button>
      {/if}
    </section>
  {/if}

  {#if later.length}
    <section>
      <h2>다음에 할 일 <span class="n">{later.length}</span></h2>
      {#each showLater ? later : later.slice(0, LATER) as t (t.id)}
        <div data-task={t.id}><TaskCard ticket={t} isNew={isNew.has(t.id)} selected={selected === t.id} actions onselect={() => onselect(t)} {onerror} /></div>
      {/each}
      {#if later.length > LATER}
        <button class="more" onclick={() => (showLater = !showLater)}>{showLater ? '접기' : `${later.length - LATER}개 더 보기`}</button>
      {/if}
    </section>
  {/if}

  {#if longTerm.length}
    <section>
      <h2>길게 보는 일 <span class="n">{longTerm.length}</span><small>조금씩 진행하는 일이라 오래 그대로여도 괜찮아요</small></h2>
      {#each longTerm as t (t.id)}
        <div data-task={t.id}><TaskCard ticket={t} isNew={isNew.has(t.id)} selected={selected === t.id} onselect={() => onselect(t)} {onerror} /></div>
      {/each}
    </section>
  {/if}

  {#if !mine.length}
    <div class="empty"><div class="big">🎉</div><p>새 일이 생기면 알려 드릴게요. 아래 칸에 직접 적어도 돼요.</p></div>
  {/if}

  {#if teamTasks.length}
    <section>
      <h2>{myArea === 'none' ? '팀은 지금' : `${areaName[myArea]} 팀은 지금`}</h2>
      <NowDoing tickets={teamTasks} {onselect} plain />
    </section>
  {/if}

  {#if others.length}
    <section>
      <h2>다른 팀</h2>
      {#each others as a (a.id)}
        <button class="team" onclick={() => go({ kind: 'area', id: a.id })}>
          <b>{areaName[a.id]}</b>
          <span>{a.people}명 · 진행 중 {a.working} · 남은 일 {a.open}{#if a.late}{' · '}<em>마감 지남 {a.late}</em>{/if}</span>
          <span class="go">›</span>
        </button>
      {/each}
    </section>
  {/if}
</div>

<style>
  .today { padding: 2px 4px 16px; }
  .headline { margin: 2px 6px 6px; font-size: 15px; font-weight: 700; display: flex; align-items: baseline; gap: 10px; }
  .headline .late { font-size: 12px; font-weight: 600; color: var(--red); }
  .headline .done { font-size: 12px; font-weight: 600; color: var(--green); }
  section { margin-top: 14px; }
  h2 { margin: 0 6px 4px; font-size: 12px; font-weight: 700; color: var(--fg3); }
  h2 .n { font-weight: 500; }
  h2 small { margin-left: 8px; font-size: 11px; font-weight: 500; }
  .ask, .team { width: 100%; display: flex; align-items: center; gap: 10px; padding: 10px 12px; margin-bottom: 4px; border-radius: 10px; background: var(--fill); text-align: left; }
  .ask:hover, .team:hover { background: var(--fill-hover); }
  .ask .dot { width: 26px; height: 26px; flex: none; border-radius: 50%; display: grid; place-items: center; background: color-mix(in srgb, var(--accent) 16%, transparent); color: var(--accent); font-size: 13px; }
  .ask .text { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
  .ask b { font-size: 13px; }
  .ask small { font-size: 11.5px; color: var(--fg3); }
  .go { flex: none; font-size: 12px; font-weight: 600; color: var(--accent); }
  .team { padding: 8px 12px; gap: 12px; }
  .team b { width: 52px; flex: none; font-size: 13px; }
  .team span:not(.go) { flex: 1; font-size: 12px; color: var(--fg2); }
  .team em { font-style: normal; color: var(--red); font-weight: 600; }
  .more { margin: 2px 6px; padding: 4px 8px; border-radius: 7px; font-size: 12px; font-weight: 600; color: var(--fg2); }
  .more:hover { background: var(--fill); }
  .empty { text-align: center; padding: 30px 20px 10px; color: var(--fg3); font-size: 12.5px; }
  .empty .big { font-size: 30px; }
</style>
