<script lang="ts">
  // Task input shared by the menu bar list and the quick add panel.
  // "@민수 배너 수정 #나루 금요일까지" → assignee, project and deadline, with autocomplete.
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import type { CreateInput } from '../../bindings/github.com/wondde/betty/internal/berry'
  import { activeToken, insertSuggestion, parseComposer, suggestions, type Named, type Token } from '../lib/compose'
  import { app, message } from '../lib/view.svelte'

  let {
    big = false,
    autofocus = false,
    onresult = (_text: string, _ok: boolean) => {},
    onlayout = (_extra: boolean) => {},
    oncancel = () => {},
  }: {
    big?: boolean
    autofocus?: boolean
    onresult?: (text: string, ok: boolean) => void
    onlayout?: (extra: boolean) => void
    oncancel?: () => void
  } = $props()

  let value = $state('')
  let caret = $state(0)
  let index = $state(0)
  let busy = $state(false)
  let input: HTMLInputElement

  const team = $derived({ members: app.view.members ?? [], projects: app.view.projects ?? [] })
  const token: Token | null = $derived(activeToken(value, caret))
  const items: (Named & { color?: string })[] = $derived(suggestions(token, team))
  const parsed = $derived(value.trim() ? parseComposer(value, team) : null)
  const me = $derived(app.view.member?.id)
  const forOther = $derived(!!parsed?.assignee && parsed.assignee.id !== me)
  const chips = $derived.by(() => {
    const out: { tone: string; label: string }[] = []
    if (!parsed) return out
    if (forOther) out.push({ tone: 'to', label: `담당 ${parsed.assignee!.name}` })
    else if (parsed.project || parsed.due) out.push({ tone: '', label: '내 할 일' })
    if (parsed.project) out.push({ tone: '', label: `#${parsed.project.name}` })
    if (parsed.due) {
      const [, m, d] = parsed.due.split('-')
      out.push({ tone: '', label: `${Number(m)}/${Number(d)} 마감` })
    }
    for (const p of parsed.problems) out.push({ tone: 'problem', label: `${p.token}을(를) 찾지 못했어요` })
    return out
  })
  const showList = $derived(items.length > 0)

  $effect(() => onlayout(showList || chips.length > 0))
  $effect(() => {
    if (autofocus) input?.focus()
  })

  export function focus() {
    input?.focus()
  }

  function sync() {
    caret = input.selectionStart ?? value.length
    if (index >= items.length) index = 0
  }

  function pick(i: number) {
    if (!token || !items[i]) return
    const next = insertSuggestion(value, token, items[i])
    value = next.text
    index = 0
    queueMicrotask(() => {
      input.setSelectionRange(next.caret, next.caret)
      caret = next.caret
    })
  }

  async function submit() {
    if (!parsed || busy) return
    if (parsed.problems.length) return onresult(`${parsed.problems[0].token}을(를) 찾지 못했어요`, false)
    if (!parsed.title) return onresult('무슨 일인지 적어 주세요', false)
    const payload: CreateInput = { title: parsed.title }
    if (parsed.assignee) payload.assignee_id = parsed.assignee.id
    if (parsed.project) payload.project_id = parsed.project.id
    if (parsed.due) payload.due_date = parsed.due
    busy = true
    try {
      const t = await Betty.Create(payload)
      value = ''
      caret = 0
      onresult(forOther ? `${parsed.assignee!.name}님에게 BT-${t.number}을 넘겼어요` : `BT-${t.number} 추가했어요`, true)
    } catch (e) {
      onresult(message(e), false)
    } finally {
      busy = false
    }
  }

  function keydown(e: KeyboardEvent) {
    if (e.isComposing || e.keyCode === 229) return // Korean IME composition
    if (showList && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      e.preventDefault()
      index = (index + (e.key === 'ArrowDown' ? 1 : items.length - 1)) % items.length
    } else if (showList && (e.key === 'Enter' || e.key === 'Tab')) {
      e.preventDefault()
      pick(index)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      submit()
    } else if (e.key === 'Escape') {
      if (value) value = ''
      else oncancel()
    }
  }
</script>

<div class="composer" class:big>
  <div class="field">
    <span class="plus" aria-hidden="true">＋</span>
    <input
      bind:this={input}
      bind:value
      oninput={sync}
      onclick={sync}
      onkeyup={sync}
      onkeydown={keydown}
      placeholder={big ? '할 일 추가 — @이름 #프로젝트 내일' : '할 일 추가 · @이름 #프로젝트 내일'}
      aria-label="할 일 추가"
      spellcheck="false"
      autocomplete="off"
      disabled={busy}
    />
    {#if busy}<span class="spinner" aria-label="추가하는 중"></span>{/if}
  </div>
  {#if showList}
    <ul class="suggest" role="listbox">
      {#each items as item, i (item.id)}
        <li role="option" aria-selected={i === index} class:active={i === index} onpointerdown={(e) => (e.preventDefault(), pick(i))}>
          {#if token?.kind === 'project'}<i style:background={item.color || '#8e8e93'}></i>{:else}<span class="at">@</span>{/if}
          {item.name}
        </li>
      {/each}
    </ul>
  {:else if chips.length}
    <div class="chips">
      {#each chips as c}<span class="chip {c.tone}">{c.label}</span>{/each}
    </div>
  {/if}
</div>

<style>
  .composer { display: flex; flex-direction: column-reverse; gap: 6px; } /* in the list window, suggestions open upward */
  .composer.big { flex-direction: column; }
  .field {
    display: flex; align-items: center; gap: 8px; height: 34px; padding: 0 10px;
    border-radius: 10px; background: var(--fill); box-shadow: inset 0 0 0 0.5px var(--line);
  }
  .field:focus-within { box-shadow: inset 0 0 0 1.5px var(--accent); }
  .plus { color: var(--fg3); font-size: 15px; }
  input { flex: 1; min-width: 0; border: none; outline: none; background: transparent; font-size: 13px; }
  input::placeholder { color: var(--fg3); }
  .big .field { height: 48px; padding: 0 16px; border-radius: 14px; background: transparent; box-shadow: none; }
  .big .plus { font-size: 20px; }
  .big input { font-size: 19px; }
  .suggest { list-style: none; margin: 0; padding: 4px; border-radius: 10px; background: var(--fill); }
  .big .suggest { margin: 0 8px 8px; }
  li { display: flex; align-items: center; gap: 7px; padding: 6px 9px; border-radius: 7px; font-weight: 500; }
  li.active { background: var(--accent); color: #fff; }
  li i { width: 8px; height: 8px; border-radius: 50%; }
  .at { color: var(--fg3); }
  li.active .at { color: rgba(255,255,255,0.8); }
  .chips { display: flex; flex-wrap: wrap; gap: 5px; padding: 0 2px; }
  .big .chips { padding: 0 16px 10px; }
  .chip { padding: 2px 8px; border-radius: 6px; font-size: 11px; font-weight: 600; background: var(--fill); color: var(--fg2); }
  .chip.to { color: var(--accent); }
  .chip.problem { color: var(--red); }
  .spinner { width: 14px; height: 14px; border-radius: 50%; border: 2px solid var(--fg3); border-top-color: transparent; animation: spin 0.7s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
