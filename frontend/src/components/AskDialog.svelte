<script lang="ts">
  // The dialog for ask(): Return confirms, Esc or a click outside cancels.
  import { asking, answer } from '../lib/ask.svelte'
  const focus = (el: HTMLElement) => { el.focus() }
</script>

<svelte:window onkeydown={(e) => {
  if (!asking.current) return
  if (e.key === 'Escape') { e.stopPropagation(); answer(false) }
  else if (e.key === 'Enter' && !e.isComposing) { e.preventDefault(); answer(true) }
}} />

{#if asking.current}
  <div class="backdrop" role="presentation" onclick={() => answer(false)}></div>
  <div class="dialog" role="alertdialog" aria-modal="true" aria-label={asking.current.message}>
    <p>{asking.current.message}</p>
    <div class="actions">
      <button onclick={() => answer(false)}>취소</button>
      <button class="ok" class:danger={asking.current.danger} onclick={() => answer(true)} use:focus>{asking.current.ok}</button>
    </div>
  </div>
{/if}

<style>
  .backdrop { position: fixed; inset: 0; z-index: 30; background: rgba(0, 0, 0, 0.2); }
  .dialog {
    position: fixed; z-index: 31; left: 50%; top: 30%; transform: translate(-50%, -50%); width: min(380px, calc(100% - 40px));
    padding: 18px; border-radius: 14px; background: var(--sheet, #fff); color: var(--fg);
    box-shadow: 0 18px 50px rgba(0, 0, 0, 0.3), 0 0 0 0.5px var(--line);
  }
  @media (prefers-color-scheme: dark) { .dialog { --sheet: #2c2c2e; } }
  p { margin: 0 0 14px; font-size: 13px; line-height: 1.5; white-space: pre-wrap; }
  .actions { display: flex; justify-content: flex-end; gap: 8px; }
  button { height: 28px; padding: 0 14px; border-radius: 8px; background: var(--fill); font-weight: 600; font-size: 12.5px; }
  button:hover { background: var(--fill-hover); }
  .ok { background: var(--accent); color: #fff; }
  .ok.danger { background: var(--red); }
  .ok:hover { filter: brightness(1.08); background: var(--accent); }
  .ok.danger:hover { background: var(--red); }
</style>
