<script lang="ts">
  // Spotlight-style panel (⌥⌘N). Grows when suggestions or the preview show,
  // hides itself after a task is added.
  import { onMount } from 'svelte'
  import { Events, Window } from '@wailsio/runtime'
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { BettyQuickOpen } from '../../bindings/events'
  import { app } from '../lib/view.svelte'
  import Composer from './Composer.svelte'

  let composer = $state<Composer>()
  let note = $state<{ text: string; ok: boolean } | null>(null)
  let timer: ReturnType<typeof setTimeout>

  const hide = () => Betty.HideWindow('quick')

  onMount(() =>
    Events.On(BettyQuickOpen, () => {
      note = null
      composer?.focus()
    }),
  )

  function layout(extra: boolean) {
    Window.SetSize(620, extra || note ? 230 : 64)
  }

  function result(text: string, ok: boolean) {
    note = { text, ok }
    layout(true)
    clearTimeout(timer)
    if (ok) timer = setTimeout(() => (hide(), (note = null), layout(false)), 1100)
  }
</script>

<div class="panel">
  {#if !app.view.signedIn}
    <div class="signed-out">
      <span>메뉴바의 Betty에서 먼저 로그인해 주세요.</span>
      <button onclick={hide}>닫기</button>
    </div>
  {:else}
    <Composer bind:this={composer} big autofocus onresult={result} onlayout={layout} oncancel={hide} />
    {#if note}<p class="note" class:ok={note.ok}>{note.ok ? '✓ ' : ''}{note.text}</p>{/if}
  {/if}
</div>

<style>
  .panel { padding: 8px; height: 100%; }
  .note { margin: 2px 16px 8px; font-size: 12px; font-weight: 600; color: var(--red); }
  .note.ok { color: var(--green); }
  .signed-out { display: flex; align-items: center; justify-content: space-between; height: 48px; padding: 0 16px; font-size: 14px; color: var(--fg2); }
  .signed-out button { padding: 4px 10px; border-radius: 7px; background: var(--fill); }
</style>
