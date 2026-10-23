<script lang="ts">
  // 정리하기: open tasks nobody owns or nobody touched for a week, one at a time, oldest
  // first. Each gets one decision by key: 1 내가 할게 · 2 담당 지정 · 3 완료 · 4 휴지통 ·
  // → 건너뛰기 (← back). The list is fixed when the mode opens so it doesn't shift under you.
  import { onMount } from 'svelte'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'
  import { memberArea, areaName } from '../lib/nav.svelte'
  import { dueOf, needsCleanup } from '../lib/tasks'

  let { project = '', onclose, onsaved, onerror }: { project?: string; onclose: () => void; onsaved: (m: string) => void; onerror: (m: string) => void } = $props()

  let queue = $state<string[]>([])
  let index = $state(0)
  let busy = $state(false)
  let picking = $state(false)
  const decided = $state<Record<string, string>>({})

  onMount(() => {
    queue = (app.view.tickets ?? [])
      .filter((t) => needsCleanup(t) && (!project || t.project?.id === project))
      .sort((a, b) => a.updated_at.localeCompare(b.updated_at))
      .map((t) => t.id)
  })

  const t = $derived(app.view.tickets?.find((x) => x.id === queue[index]))
  const me = $derived(app.view.member?.id)
  const owner = $derived(app.view.members?.find((m) => m.id === t?.assignee_id))
  const checklist = $derived.by(() => {
    const p = t ? app.view.progress?.[t.id] : undefined
    const done = p?.finished?.length ?? 0
    return { done, total: done + (p?.remaining?.length ?? 0) }
  })
  const age = (iso?: string) => {
    if (!iso) return ''
    const days = Math.floor((Date.now() - Date.parse(iso)) / 86400000)
    return days < 1 ? '오늘' : `${days}일 전`
  }
  const left = $derived(queue.length - Object.keys(decided).length)

  function next() {
    picking = false
    if (index < queue.length - 1) index++
    else index = queue.length // done screen
  }
  async function act(label: string, fn: () => Promise<unknown>) {
    if (!t || busy) return
    const id = t.id
    busy = true
    try {
      await fn()
      decided[id] = label
      next()
    } catch (e) {
      onerror(message(e))
    } finally {
      busy = false
    }
  }
  const claim = () => act('내가 맡음', () => Betty.Claim(t!.id))
  const assign = (id: string) => act(`${app.view.members?.find((m) => m.id === id)?.name}에게`, () => Betty.Update(t!.id, { assignee_id: id }))
  const finish = () => act('완료', () => Betty.SetStatus(t!.id, 'done'))
  const trash = () => act('휴지통', () => Betty.Delete(t!.id))

  function keydown(e: KeyboardEvent) {
    if ((e.target as HTMLElement).matches('input, textarea, select')) return
    if (e.key === 'Escape') return picking ? (picking = false) : onclose()
    if (!t) return
    if (e.key === '1') claim()
    else if (e.key === '2') picking = !picking
    else if (e.key === '3') finish()
    else if (e.key === '4') trash()
    else if (e.key === 'ArrowRight' || e.key === ' ') (e.preventDefault(), next())
    else if (e.key === 'ArrowLeft') index = Math.max(0, index - 1)
  }
  const members = $derived(
    [...(app.view.members ?? [])].sort((a, b) => (t?.area && memberArea(a) === t.area ? -1 : 0) - (t?.area && memberArea(b) === t.area ? -1 : 0) || a.name.localeCompare(b.name)),
  )
</script>

<svelte:window onkeydown={keydown} />

<div class="cleanup" role="dialog" aria-modal="true" aria-label="정리하기">
  <div class="drag" aria-hidden="true"></div>
  <header>
    <div>
      <h1>정리하기</h1>
      <p>아무도 안 맡았거나 일주일 넘게 그대로인 일을 하나씩 정리해요. 남은 일 {left}개</p>
    </div>
    <button class="close" onclick={onclose}>끝내기 (Esc)</button>
  </header>
  <div class="bar"><i style:width="{queue.length ? (Object.keys(decided).length / queue.length) * 100 : 100}%"></i></div>

  {#if t}
    <article>
      <div class="meta">
        <span class="num">BT-{t.number} · {index + 1}/{queue.length}</span>
        {#if t.project}<span class="project" style:--pc={t.project.color}>{t.project.name}</span>{/if}
        {#if t.area}<span>{areaName[memberArea({ area: t.area })]}</span>{/if}
        <span>{t.status === 'in_progress' ? '진행 중' : t.status === 'blocked' ? '막힘' : '할 일'}</span>
        {#if dueOf(t)}<span class="due">{dueOf(t)!.label}</span>{/if}
        {#if checklist.total}<span>☑ {checklist.done}/{checklist.total}</span>{/if}
      </div>
      <h2>{t.title}</h2>
      {#if t.description}<p class="desc">{t.description}</p>{/if}
      <p class="when">
        {owner ? `${owner.name}님 담당` : '맡은 사람 없음'} · 만든 지 {age(t.created_at)} · 마지막 변경 {age(t.updated_at)}
        {#if t.source_url} · <button class="link" onclick={() => Betty.OpenURL(t.source_url)}>대화 보기 ↗</button>{/if}
      </p>
      {#if decided[t.id]}<p class="done">이미 정리함: {decided[t.id]}</p>{/if}
    </article>

    <div class="actions">
      {#if t.assignee_id !== me}<button disabled={busy} onclick={claim}><kbd>1</kbd>내가 할게</button>{/if}
      <button disabled={busy} class:on={picking} onclick={() => (picking = !picking)}><kbd>2</kbd>담당 지정</button>
      <button disabled={busy} onclick={finish}><kbd>3</kbd>이미 끝났어요</button>
      <button disabled={busy} class="danger" onclick={trash}><kbd>4</kbd>필요 없어요</button>
      <button disabled={busy} class="skip" onclick={next}>건너뛰기 <kbd>→</kbd></button>
    </div>
    {#if picking}
      <div class="people" role="listbox" aria-label="담당자">
        {#each members as m (m.id)}
          <button onclick={() => assign(m.id)} class:same={!!t.area && memberArea(m) === t.area}><span class="avatar">{Array.from(m.name)[0]}</span>{m.name}<small>{areaName[memberArea(m)]}</small></button>
        {/each}
      </div>
    {/if}
  {:else if index < queue.length}
    <article><p class="done">휴지통으로 옮긴 일이에요.</p></article>
    <div class="actions"><button class="skip" onclick={next}>건너뛰기 <kbd>→</kbd></button></div>
  {:else}
    <div class="empty">
      <div class="big">✨</div>
      <p><b>{queue.length ? `${Object.keys(decided).length}개를 정리했어요` : '정리할 일이 없어요'}</b></p>
      <p>{queue.length ? '남은 일은 다음에 다시 정리하기에서 볼 수 있어요.' : '모든 일에 담당자가 있고 최근에 움직였어요.'}</p>
      <button class="close" onclick={onclose}>돌아가기</button>
    </div>
  {/if}
</div>

<style>
  .cleanup { position: fixed; inset: 0; z-index: 20; display: flex; flex-direction: column; padding: 46px 40px 24px; background: var(--settings-bg, #f5f5f7); color: var(--fg); }
  @media (prefers-color-scheme: dark) { .cleanup { --settings-bg: #1e1e20; } }
  .drag { position: absolute; top: 0; left: 0; right: 0; height: 40px; --wails-draggable: drag; }
  header { display: flex; align-items: flex-end; justify-content: space-between; gap: 12px; }
  h1 { margin: 0; font-size: 20px; }
  header p { margin: 2px 0 0; font-size: 12px; color: var(--fg3); }
  .close { padding: 5px 12px; border-radius: 8px; background: var(--fill); font-weight: 600; font-size: 12.5px; }
  .close:hover { background: var(--fill-hover); }
  .bar { height: 3px; margin: 12px 0 18px; border-radius: 2px; background: var(--fill); overflow: hidden; }
  .bar i { display: block; height: 100%; background: var(--accent); transition: width 0.2s ease; }
  article { max-width: 720px; width: 100%; margin: 0 auto; padding: 20px 22px; border-radius: 14px; background: var(--card-solid, #fff); box-shadow: 0 1px 3px rgba(0,0,0,0.1), 0 0 0 0.5px var(--line); }
  @media (prefers-color-scheme: dark) { article { --card-solid: #2c2c2e; } }
  .meta { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; font-size: 11.5px; color: var(--fg3); }
  .num { font-weight: 700; font-variant-numeric: tabular-nums; }
  .project { padding: 0 6px; border-radius: 5px; font-weight: 600; color: color-mix(in srgb, var(--pc) 80%, var(--fg)); background: color-mix(in srgb, var(--pc) 16%, transparent); }
  .due { color: var(--red); font-weight: 600; }
  h2 { margin: 10px 0 6px; font-size: 18px; line-height: 1.35; }
  .desc { margin: 0; max-height: 180px; overflow-y: auto; font-size: 13px; line-height: 1.55; color: var(--fg2); white-space: pre-wrap; user-select: text; -webkit-user-select: text; }
  .when { margin: 12px 0 0; font-size: 12px; color: var(--fg3); }
  .link { color: var(--accent); font-weight: 600; }
  .done { margin: 8px 0 0; font-size: 12px; color: var(--green); font-weight: 600; }
  .actions { display: flex; flex-wrap: wrap; justify-content: center; gap: 8px; margin: 18px auto 0; max-width: 720px; }
  .actions button { display: inline-flex; align-items: center; gap: 6px; height: 34px; padding: 0 14px; border-radius: 9px; background: var(--fill); font-weight: 600; }
  .actions button:hover, .actions button.on { background: var(--fill-hover); }
  .actions .danger { color: var(--red); }
  .actions .skip { color: var(--fg2); }
  kbd { min-width: 18px; padding: 0 4px; border-radius: 4px; background: var(--fill-hover); font: 600 11px -apple-system, sans-serif; color: var(--fg2); text-align: center; }
  button:disabled { opacity: 0.5; }
  .people { display: flex; flex-wrap: wrap; justify-content: center; gap: 6px; margin: 12px auto 0; max-width: 720px; }
  .people button { display: inline-flex; align-items: center; gap: 6px; height: 30px; padding: 0 10px 0 4px; border-radius: 15px; background: var(--fill); font-size: 12.5px; }
  .people button:hover { background: var(--accent); color: #fff; }
  .people button.same { box-shadow: inset 0 0 0 1px var(--accent); }
  .people small { color: var(--fg3); font-size: 10.5px; }
  .people button:hover small { color: rgba(255,255,255,0.8); }
  .avatar { width: 22px; height: 22px; border-radius: 50%; background: var(--fill-hover); display: grid; place-items: center; font-size: 10.5px; font-weight: 700; }
  .empty { margin: auto; text-align: center; color: var(--fg3); line-height: 1.6; }
  .empty .big { font-size: 34px; }
  .empty b { color: var(--fg); font-size: 15px; }
  .empty .close { margin-top: 12px; }
</style>
