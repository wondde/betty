<script lang="ts">
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'

  let error = $state('')

  async function start() {
    error = ''
    try {
      await Betty.SignIn()
    } catch (e) {
      error = message(e)
    }
  }
</script>

<section class="signin">
  <img class="berry" src="/betty-icon.png" alt="" width="88" height="88" />
  <h1>Betty</h1>
  <p>Discord 계정으로 로그인하면<br />나에게 배정된 업무가 메뉴바에 모여요.</p>
  {#if app.view.signingIn}
    <p class="wait">브라우저에서 로그인을 마치면 자동으로 돌아와요.</p>
    <button class="ghost" onclick={() => Betty.CancelSignIn()}>취소</button>
  {:else}
    <button class="discord" onclick={start}>Discord로 로그인</button>
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <footer>{app.view.server?.replace(/^https?:\/\//, '')}</footer>
</section>

<style>
  .signin { height: 100%; padding-top: 40px !important; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 24px 28px; text-align: center; }
  .berry { width: 88px; height: 88px; margin-bottom: 2px; }
  h1 { margin: 0; font-size: 22px; letter-spacing: -0.3px; }
  p { margin: 0; color: var(--fg2); line-height: 1.5; }
  .discord { margin-top: 8px; width: 100%; height: 36px; border-radius: 10px; background: var(--discord); color: #fff; font-weight: 700; }
  .discord:hover { filter: brightness(1.08); }
  .ghost { padding: 4px 12px; border-radius: 7px; background: var(--fill); }
  .wait { font-size: 12px; }
  .error { color: var(--red); font-size: 12px; }
  footer { position: absolute; bottom: 14px; font-size: 11px; color: var(--fg3); }
</style>
