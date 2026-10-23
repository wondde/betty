<script lang="ts">
  // 대화로 할 일 만들기 (⇧⌘N): paste a conversation from anywhere (KakaoTalk, meeting notes,
  // another Discord server) and Betty's extractor turns it into tasks. What it finds lands in
  // "베티가 찾은 일" unless automatic tasks are on and the request is clear.
  import { onMount } from 'svelte'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'
  import { nav } from '../lib/nav.svelte'

  let { onclose, onsent }: { onclose: () => void; onsent: (m: string) => void } = $props()

  let content = $state('')
  let project = $state(nav.scope.kind === 'project' ? nav.scope.id : '')
  let busy = $state(false)
  let error = $state('')
  let field = $state<HTMLTextAreaElement>()

  const length = $derived([...content.trim()].length)
  onMount(() => field?.focus())

  async function send() {
    if (!content.trim() || busy) return
    busy = true
    error = ''
    try {
      await Betty.Capture(content, project)
      onsent('베티에게 정리를 맡겼어요. 찾은 일은 ‘베티가 찾은 일’에 올라와요')
      onclose()
    } catch (e) {
      error = message(e)
    } finally {
      busy = false
    }
  }

  function keydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose()
    else if (e.key === 'Enter' && e.metaKey) send()
  }
</script>

<div class="backdrop" role="presentation" onclick={onclose}></div>
<div class="sheet" role="dialog" aria-modal="true" aria-label="대화로 할 일 만들기" onkeydown={keydown} tabindex="-1">
  <h1>대화로 할 일 만들기</h1>
  <p class="muted">다른 곳에서 나눈 대화나 회의 메모를 붙여 넣으면 베티가 할 일을 찾아 정리해요.</p>
  <textarea bind:this={field} bind:value={content} placeholder="여기에 붙여 넣으세요 (⌘V)" spellcheck="false"></textarea>
  <div class="row">
    <label>
      <span class="muted">프로젝트</span>
      <select bind:value={project}>
        <option value="">베티가 알아서 정하기</option>
        {#each app.view.projects ?? [] as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
      </select>
    </label>
    <span class="count" class:over={length > 40000}>{length.toLocaleString()} / 40,000자</span>
  </div>
  {#if error}<p class="error">{error}</p>{/if}
  <div class="actions">
    <button onclick={onclose}>취소</button>
    <button class="primary" disabled={!content.trim() || busy || length > 40000} onclick={send}>{busy ? '보내는 중…' : '정리 맡기기 (⌘↩)'}</button>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.18); z-index: 10; }
  .sheet {
    position: fixed; left: 50%; top: 52px; width: min(560px, calc(100% - 32px)); transform: translateX(-50%); z-index: 11;
    display: flex; flex-direction: column; gap: 10px; padding: 18px; border-radius: 14px; outline: none;
    background: var(--sheet, #fff); color: var(--fg); box-shadow: 0 18px 50px rgba(0, 0, 0, 0.25), 0 0 0 0.5px var(--line);
  }
  @media (prefers-color-scheme: dark) { .sheet { --sheet: #2c2c2e; } }
  h1 { margin: 0; font-size: 17px; }
  p { margin: 0; font-size: 12.5px; }
  .muted { color: var(--fg3); }
  textarea {
    height: 260px; resize: vertical; padding: 10px 12px; border: none; outline: none; border-radius: 9px;
    background: var(--fill); font: inherit; font-size: 12.5px; line-height: 1.55; color: inherit; user-select: text; -webkit-user-select: text;
  }
  textarea:focus, select:focus { box-shadow: inset 0 0 0 1.5px var(--accent); }
  .row { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
  label { display: flex; align-items: center; gap: 8px; font-size: 12.5px; }
  select { height: 26px; padding: 0 8px; border: none; outline: none; border-radius: 7px; background: var(--fill); font: inherit; font-size: 12.5px; color: inherit; }
  .count { font-size: 11px; color: var(--fg3); font-variant-numeric: tabular-nums; }
  .count.over, .error { color: var(--red); }
  .error { font-size: 12px; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; }
  .actions button { padding: 5px 14px; border-radius: 8px; background: var(--fill); font-weight: 600; }
  .actions button:hover { background: var(--fill-hover); }
  .actions .primary { background: var(--accent); color: #fff; }
  .actions .primary:disabled { opacity: 0.45; }
</style>
