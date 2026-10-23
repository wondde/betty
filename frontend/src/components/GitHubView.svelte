<script lang="ts">
  // 개발 현황: issues and PRs from the connected repositories, which task each belongs to,
  // and the assistant's "same work?" suggestions to confirm or reject.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'
  import { nav } from '../lib/nav.svelte'
  import { stageLabel } from '../lib/tasks'

  let { onerror, onsaved }: { onerror: (m: string) => void; onsaved: (m: string) => void } = $props()

  let status = $state<'open' | 'done' | 'all'>('open')
  let kind = $state<'all' | 'issue' | 'pr'>('all')
  let project = $state('')
  let query = $state('')
  let limit = $state(80)
  let linking = $state<string | null>(null) // item key with the task picker open
  let pick = $state('')
  let busy = $state(false)

  const v = $derived(app.view)
  const gh = $derived(v.github ?? { repos: [], items: [], suggestions: [], identities: [] })
  const key = (i: { repo_id: string; number: number }) => `${i.repo_id}:${i.number}`
  const repoName = (id: string) => gh.repos?.find((r) => r.id === id)?.full_name?.split('/').pop() ?? ''
  const person = (login: string) => {
    const id = gh.identities?.find((x) => x.login.toLowerCase() === login.toLowerCase())?.member_id
    return v.members?.find((m) => m.id === id)?.name ?? login
  }
  const ticket = (id: string) => v.tickets?.find((t) => t.id === id)

  const items = $derived.by(() => {
    const q = query.trim().toLowerCase()
    return (gh.items ?? [])
      .filter((i) => (status === 'all' || (status === 'open' ? i.state === 'open' : i.state === 'closed')) && (kind === 'all' || i.kind === kind) && (!project || i.project_id === project))
      .filter((i) => !q || i.title.toLowerCase().includes(q) || String(i.number) === q.replace('#', '') || i.author.includes(q) || i.assignees?.some((a) => a.includes(q)))
      .sort((a, b) => b.updated_at.localeCompare(a.updated_at))
  })
  const suggestions = $derived((gh.suggestions ?? []).filter((s) => ticket(s.ticket_id) && gh.items?.some((i) => key(i) === key(s))))
  // One card per task: the assistant often proposes several PRs for the same task with the same reason.
  const byTask = $derived.by(() => {
    const m = new Map<string, typeof suggestions>()
    for (const s of suggestions) m.has(s.ticket_id) ? m.get(s.ticket_id)!.push(s) : m.set(s.ticket_id, [s])
    return [...m.entries()].map(([id, list]) => ({ t: ticket(id)!, list, reasons: [...new Set(list.map((s) => s.reason).filter(Boolean))] }))
  })
  let reasonOpen = $state<Record<string, boolean>>({})
  async function linkAll(list: typeof suggestions) {
    await run(async () => {
      for (const s of list) await Betty.LinkGitHub(s.ticket_id, s.repo_id, s.number, false, false)
    }, list.length > 1 ? `${list.length}개를 한 업무로 연결했어요` : '한 업무로 연결했어요')
  }
  const candidates = $derived.by(() => {
    const item = gh.items?.find((i) => key(i) === linking)
    return (v.tickets ?? []).filter((t) => t.status !== 'done' && (!item || t.project?.id === item.project_id)).sort((a, b) => b.number - a.number)
  })

  async function run(fn: () => Promise<void>, note: string) {
    busy = true
    try {
      await fn()
      onsaved(note)
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = false
    }
  }
  function openTicket(id: string) {
    nav.scope = { kind: 'team' }
    nav.selected = id
  }
</script>

<div class="gh">
  {#if suggestions.length}
    <section class="suggest">
      <h3>같은 업무인지 확인해 주세요 <span>{suggestions.length}</span></h3>
      {#each byTask.slice(0, 20) as g (g.t.id)}
        <article class="pair">
          <div class="task"><small>할 일</small><button class="tlink" onclick={() => openTicket(g.t.id)}>BT-{g.t.number} {g.t.title}</button></div>
          <ul class="items">
            {#each g.list as s (key(s))}
              {@const i = gh.items!.find((x) => key(x) === key(s))!}
              <li>
                <button class="ilink" onclick={() => Betty.OpenURL(i.url)} title="GitHub에서 보기"><small>{i.kind === 'pr' ? 'PR' : '이슈'} #{i.number}</small> {i.title}</button>
                <span class="row-actions">
                  {#if g.list.length > 1}<button disabled={busy} onclick={() => run(() => Betty.LinkGitHub(s.ticket_id, s.repo_id, s.number, false, false), '한 업무로 연결했어요')} title="이것만 연결">✓</button>{/if}
                  <button disabled={busy} onclick={() => run(() => Betty.RejectGitHub(s.ticket_id, s.repo_id, s.number), '다른 업무로 기억할게요')} title="다른 업무예요">✕</button>
                </span>
              </li>
            {/each}
          </ul>
          {#if g.reasons.length}
            <button class="why" onclick={() => (reasonOpen[g.t.id] = !reasonOpen[g.t.id])}>{reasonOpen[g.t.id] ? '이유 접기' : '왜 같은 업무 같나요?'}</button>
            {#if reasonOpen[g.t.id]}{#each g.reasons as r}<p class="reason">{r}</p>{/each}{/if}
          {/if}
          <div class="actions">
            <button class="primary" disabled={busy} onclick={() => linkAll(g.list)}>{g.list.length > 1 ? `모두 같은 업무예요 (${g.list.length})` : '같은 업무예요'}</button>
            {#if g.list.length === 1}<button disabled={busy} onclick={() => run(() => Betty.RejectGitHub(g.list[0].ticket_id, g.list[0].repo_id, g.list[0].number), '다른 업무로 기억할게요')}>다른 업무예요</button>{/if}
          </div>
        </article>
      {/each}
    </section>
  {/if}

  <div class="filters">
    {#each [['open', '진행 중'], ['done', '닫힘·완료'], ['all', '전체']] as [value, label]}
      <button class:on={status === value} onclick={() => ((status = value as typeof status), (limit = 80))}>{label}</button>
    {/each}
    <span class="sep"></span>
    {#each [['all', '전체'], ['issue', '이슈'], ['pr', 'PR']] as [value, label]}
      <button class:on={kind === value} onclick={() => (kind = value as typeof kind)}>{label}</button>
    {/each}
    <select bind:value={project} aria-label="프로젝트">
      <option value="">모든 프로젝트</option>
      {#each v.projects ?? [] as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
    </select>
    <input bind:value={query} placeholder="제목·번호·사람" aria-label="GitHub 검색" spellcheck="false" />
    <span class="count">{items.length}건</span>
  </div>

  {#if !(gh.repos?.length)}
    <p class="empty">연결된 GitHub 저장소가 없어요. 팀·연결 설정에서 저장소를 연결해 주세요.</p>
  {/if}

  {#each items.slice(0, limit) as i (key(i))}
    {@const [label, tone] = stageLabel(i)}
    <article class="item" class:closed={i.state === 'closed'}>
      <span class="kind {i.kind}" title={i.kind === 'pr' ? 'PR' : '이슈'}>{i.kind === 'pr' ? '⇄' : '◉'}</span>
      <div class="main">
        <div class="title">
          <button class="titlelink" onclick={() => Betty.OpenURL(i.url)}>{i.title}</button>
          <span class="stage {tone}">{label}</span>
        </div>
        <div class="meta">
          <span>{repoName(i.repo_id)} #{i.number}</span>
          {#if i.author}<span>작성 {person(i.author)}</span>{/if}
          {#if i.assignees?.length}<span>담당 {i.assignees.map(person).join(', ')}</span>{/if}
          {#if i.reviewers?.length}<span>리뷰 {i.reviewers.map(person).join(', ')}</span>{/if}
          <span>{new Date(i.updated_at).toLocaleDateString('ko-KR', { month: 'numeric', day: 'numeric' })}</span>
        </div>
        <div class="links">
          {#each i.ticket_ids ?? [] as id (id)}
            {@const t = ticket(id)}
            {#if t}
              <span class="ticket">
                <button onclick={() => openTicket(id)}>BT-{t.number} {t.title}</button>
                <button class="x" title="연결 풀기" aria-label="BT-{t.number} 연결 풀기" disabled={busy} onclick={() => run(() => Betty.LinkGitHub(id, i.repo_id, i.number, false, true), '연결을 풀었어요')}>✕</button>
              </span>
            {/if}
          {/each}
          {#if linking === key(i)}
            <select bind:value={pick} aria-label="연결할 업무">
              <option value="">연결할 업무 선택</option>
              {#each candidates as t (t.id)}<option value={t.id}>BT-{t.number} {t.title}</option>{/each}
            </select>
            <button class="mini primary" disabled={!pick || busy} onclick={() => run(() => Betty.LinkGitHub(pick, i.repo_id, i.number, (i.ticket_ids?.length ?? 0) > 0, false), '한 업무로 연결했어요').then(() => ((linking = null), (pick = '')))}>연결</button>
            <button class="mini" onclick={() => (linking = null)}>취소</button>
          {:else}
            <button class="add" onclick={() => ((linking = key(i)), (pick = ''))}>＋ 업무 연결</button>
          {/if}
        </div>
      </div>
    </article>
  {/each}
  {#if items.length > limit}<button class="more" onclick={() => (limit += 80)}>더 보기 ({items.length - limit})</button>{/if}
</div>

<style>
  .gh { display: flex; flex-direction: column; gap: 6px; padding: 0 16px 20px; }
  .suggest { padding: 10px 12px; border-radius: 12px; background: color-mix(in srgb, var(--accent) 8%, transparent); box-shadow: 0 0 0 0.5px var(--line); margin-bottom: 6px; }
  .suggest h3 { margin: 0 0 6px; font-size: 12.5px; }
  .suggest h3 span { padding: 0 6px; border-radius: 8px; background: var(--accent); color: #fff; font-size: 11px; }
  .pair { padding: 8px 0; border-top: 0.5px solid var(--line); }
  .task small, .items small { color: var(--fg3); font-size: 11px; font-weight: 600; margin-right: 4px; }
  .tlink { font-weight: 700; font-size: 13px; text-align: left; }
  .tlink:hover { color: var(--accent); }
  .items { list-style: none; margin: 6px 0 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }
  .items li { display: flex; align-items: center; gap: 6px; padding: 3px 6px; border-radius: 7px; background: var(--fill); }
  .ilink { flex: 1; min-width: 0; text-align: left; font-size: 12.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ilink:hover { color: var(--accent); }
  .row-actions { display: flex; gap: 2px; flex: none; }
  .row-actions button { width: 22px; height: 22px; border-radius: 6px; color: var(--fg3); }
  .row-actions button:hover { background: var(--fill-hover); color: var(--fg); }
  .why { margin-top: 4px; font-size: 11.5px; color: var(--fg3); }
  .reason { margin: 4px 0; font-size: 12px; color: var(--fg2); }
  .actions, .links { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; margin-top: 4px; }
  .actions button, .mini, .add { height: 24px; padding: 0 10px; border-radius: 6px; background: var(--fill); font-size: 11.5px; font-weight: 600; }
  .primary { background: var(--accent) !important; color: #fff; }
  .link { background: transparent !important; color: var(--accent); }
  .filters { display: flex; flex-wrap: wrap; align-items: center; gap: 4px; margin: 2px 0 6px; }
  .filters button { height: 26px; padding: 0 10px; border-radius: 7px; background: var(--fill); font-size: 12px; font-weight: 600; color: var(--fg2); }
  .filters button.on { background: var(--accent); color: #fff; }
  .filters select, .filters input { height: 26px; padding: 0 8px; border-radius: 7px; border: none; outline: none; background: var(--fill); font: inherit; font-size: 12px; color: inherit; }
  .filters input { width: 140px; }
  .sep { width: 8px; }
  .count { margin-left: auto; font-size: 11.5px; color: var(--fg3); }
  .item { display: flex; gap: 10px; padding: 8px 10px; border-radius: 9px; }
  .item:hover { background: var(--fill); }
  .item.closed { opacity: 0.7; }
  .kind { width: 18px; text-align: center; font-size: 13px; color: var(--green); }
  .kind.pr { color: var(--accent); }
  .main { flex: 1; min-width: 0; }
  .title { display: flex; align-items: center; gap: 8px; }
  .titlelink { font-size: 13px; font-weight: 600; text-align: left; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .titlelink:hover { color: var(--accent); }
  .stage { flex: none; padding: 0 6px; border-radius: 5px; font-size: 10.5px; font-weight: 700; background: var(--fill); }
  .stage.merged { color: #8250df; }
  .stage.open { color: var(--green); }
  .stage.review { color: var(--orange); }
  .stage.closed, .stage.draft { color: var(--fg3); }
  .meta { display: flex; flex-wrap: wrap; gap: 2px 10px; font-size: 11px; color: var(--fg3); margin-top: 1px; }
  .ticket { display: inline-flex; align-items: center; border-radius: 6px; background: var(--fill); font-size: 11.5px; overflow: hidden; }
  .ticket button { padding: 2px 8px; max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ticket button:hover { background: var(--fill-hover); }
  .ticket .x { padding: 2px 6px; color: var(--fg3); }
  .links select { height: 24px; border: none; border-radius: 6px; background: var(--fill); font: inherit; font-size: 11.5px; color: inherit; max-width: 300px; }
  .add { color: var(--fg3); background: transparent !important; }
  .add:hover { color: var(--accent); }
  .empty { color: var(--fg3); text-align: center; padding: 40px 0; }
  .more { align-self: center; height: 28px; padding: 0 16px; border-radius: 8px; background: var(--fill); font-weight: 600; }
  button:disabled { opacity: 0.5; }
</style>
