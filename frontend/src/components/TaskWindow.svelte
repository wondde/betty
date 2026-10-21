<script lang="ts">
  // The menu bar window: my tasks grouped by urgency, recent done, and an add field.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { Ticket } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { app, message } from '../lib/view.svelte'
  import { daysUntil, dayGroup, sortTasks, todayLabel } from '../lib/tasks'
  import TaskCard from './TaskCard.svelte'
  import Composer from './Composer.svelte'
  import SignIn from './SignIn.svelte'

  let toast = $state<{ text: string; ok: boolean } | null>(null)
  let showDone = $state(false)
  let refreshing = $state(false)
  let toastTimer: ReturnType<typeof setTimeout>

  const v = $derived(app.view)
  const isNew = $derived(new Set(v.new ?? []))
  const active = $derived(sortTasks((v.tickets ?? []).filter((t) => t.status !== 'done'), isNew, v.order ?? []))
  const done = $derived((v.tickets ?? []).filter((t) => t.status === 'done').sort((a, b) => b.updated_at.localeCompare(a.updated_at)))

  // Section headings make the urgency order scannable at a glance.
  function section(t: Ticket): string {
    const days = daysUntil(t.due_date)
    if (days !== null && days <= 1) return '마감 임박'
    if (isNew.has(t.id)) return '새로 들어온 일'
    if (t.status === 'in_progress') return '진행 중'
    if (t.status === 'blocked') return '막힘'
    return '할 일'
  }
  const groups = $derived.by(() => {
    const out: { title: string; items: Ticket[] }[] = []
    for (const t of active) {
      const title = section(t)
      if (out.at(-1)?.title !== title) out.push({ title, items: [] })
      out.at(-1)!.items.push(t)
    }
    return out
  })
  const summary = $derived.by(() => {
    const urgent = active.filter((t) => (daysUntil(t.due_date) ?? 99) <= 1).length
    const working = active.filter((t) => t.status === 'in_progress').length
    return [`할 일 ${active.length}`, working && `진행 중 ${working}`, urgent && `마감 임박 ${urgent}`].filter(Boolean).join(' · ')
  })
  const synced = $derived(v.syncedAt ? new Date(v.syncedAt).toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' }) : '')

  function say(text: string, ok = false) {
    toast = { text, ok }
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => (toast = null), 3200)
  }

  async function refresh() {
    refreshing = true
    try {
      await Betty.Refresh()
    } catch (e) {
      say(message(e))
    } finally {
      refreshing = false
    }
  }
</script>

{#if !v.signedIn}
  <SignIn />
{:else}
  <div class="window">
    <header>
      <div>
        <h1>오늘</h1>
        <p class="date">{todayLabel()}</p>
      </div>
      <div class="tools">
        <button onclick={refresh} class:spin={refreshing} title={synced ? `새로고침 · ${synced} 동기화` : '새로고침'} aria-label="새로고침">
          <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9M13.5 2.5v3h-3" /></svg>
        </button>
        <button onclick={() => Betty.OpenBoard()} title="웹 보드 열기" aria-label="웹 보드 열기">
          <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M6 3H3.5v9.5H13V10M9 3h4v4M13 3L7.5 8.5" /></svg>
        </button>
      </div>
    </header>
    <p class="summary" class:urgent={summary.includes('마감 임박')}>{active.length ? summary : ''}</p>
    {#if v.offline}<p class="offline">오프라인 · 마지막으로 불러온 목록이에요</p>{/if}

    <main>
      {#if !v.hintDone && active.length}
        <div class="hint">
          <span>동그라미를 누르면 완료돼요. 잘못 눌렀다면 바로 한 번 더 누르면 취소돼요.</span>
          <button onclick={() => Betty.DismissHint()}>알겠어요</button>
        </div>
      {/if}

      {#each groups as g (g.title)}
        <h2>{g.title}</h2>
        <div class="list">
          {#each g.items as t (t.id)}
            <TaskCard ticket={t} isNew={isNew.has(t.id)} onerror={(m) => say(m)} />
          {/each}
        </div>
      {:else}
        <div class="empty">
          <div class="big">🎉</div>
          <p>모두 끝냈어요</p>
        </div>
      {/each}

      {#if done.length}
        <button class="done-toggle" onclick={() => (showDone = !showDone)}>
          {showDone ? '▾' : '▸'} 최근 완료 {done.length}
        </button>
        {#if showDone}
          <ul class="done">
            {#each done as t (t.id)}
              <li>
                <span class="tick">✓</span>
                <span class="dtitle">{t.title}</span>
                <span class="when">{dayGroup(t.updated_at)}</span>
                <button onclick={() => Betty.SetStatus(t.id, 'todo').catch((e) => say(message(e)))} title="되돌리기" aria-label="할 일로 되돌리기">↺</button>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </main>

    <footer>
      {#if toast}<p class="toast" class:ok={toast.ok}>{toast.text}</p>{/if}
      <Composer onresult={(text, ok) => say(text, ok)} />
      <p class="who">{v.member?.name} · ⌥⌘N 빠른 추가</p>
    </footer>
  </div>
{/if}

<style>
  .window { height: 100%; display: flex; flex-direction: column; }
  header { display: flex; align-items: flex-start; justify-content: space-between; padding: 16px 16px 0; --wails-draggable: drag; }
  h1 { margin: 0; font-size: 20px; font-weight: 700; letter-spacing: -0.4px; }
  .date { margin: 1px 0 0; font-size: 11.5px; color: var(--fg3); font-weight: 500; }
  .tools { display: flex; gap: 4px; --wails-draggable: no-drag; }
  .tools button { width: 26px; height: 26px; border-radius: 7px; display: grid; place-items: center; }
  .tools button:hover { background: var(--fill-hover); }
  .tools svg { width: 15px; height: 15px; fill: none; stroke: var(--fg2); stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
  .spin svg { animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .summary { margin: 6px 16px 0; min-height: 15px; font-size: 11.5px; font-weight: 600; color: var(--fg2); }
  .summary.urgent { color: var(--orange); }
  .offline { margin: 8px 16px 0; padding: 5px 10px; border-radius: 8px; background: var(--fill); font-size: 11px; color: var(--fg2); }
  main { flex: 1; overflow-y: auto; padding: 4px 12px 8px; }
  h2 { margin: 12px 4px 6px; font-size: 11px; font-weight: 700; color: var(--fg3); letter-spacing: 0.2px; }
  .list { display: flex; flex-direction: column; gap: 6px; }
  .hint { display: flex; align-items: center; gap: 8px; margin: 8px 0 0; padding: 8px 10px; border-radius: 10px; background: var(--fill); font-size: 11.5px; line-height: 1.45; color: var(--fg2); }
  .hint span { flex: 1; }
  .hint button { flex: none; padding: 3px 8px; border-radius: 6px; background: var(--fill-hover); font-size: 11px; font-weight: 700; }
  .empty { text-align: center; padding: 60px 0 40px; color: var(--fg3); }
  .empty .big { font-size: 34px; }
  .done-toggle { margin: 14px 4px 4px; font-size: 11.5px; font-weight: 700; color: var(--fg3); }
  .done { list-style: none; margin: 0; padding: 0 4px; }
  .done li { display: flex; align-items: center; gap: 8px; padding: 6px 0; border-bottom: 0.5px solid var(--line); font-size: 12px; color: var(--fg2); }
  .tick { color: var(--green); font-weight: 700; }
  .dtitle { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-decoration: line-through; text-decoration-color: var(--fg3); }
  .when { font-size: 10.5px; color: var(--fg3); }
  .done button { width: 20px; height: 20px; border-radius: 5px; color: var(--fg3); }
  .done button:hover { background: var(--fill-hover); color: var(--fg); }
  footer { position: relative; padding: 8px 12px 10px; border-top: 0.5px solid var(--line); }
  .who { margin: 6px 2px 0; font-size: 10.5px; color: var(--fg3); }
  .toast { position: absolute; left: 12px; right: 12px; bottom: calc(100% + 6px); margin: 0; padding: 8px 12px; border-radius: 10px; background: rgba(30, 30, 32, 0.92); color: #fff; font-size: 12px; font-weight: 500; }
  .toast.ok { background: color-mix(in srgb, var(--green) 85%, black); }
</style>
