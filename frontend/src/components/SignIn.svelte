<script lang="ts">
  import { Betty } from '../../bindings/github.com/wondde/betty'
  import { app, message } from '../lib/view.svelte'
  import { isWeb } from '../lib/platform'

  // The web also takes the board's name + password login (for people without Discord sign-in).
  let name = $state('')
  let password = $state('')
  let busy = $state(false)
  async function passwordLogin(e: SubmitEvent) {
    e.preventDefault()
    error = ''
    busy = true
    try {
      const res = await fetch('/api/login', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: name, password }) })
      if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || '로그인하지 못했어요')
      await Betty.Refresh()
    } catch (err) {
      error = message(err)
    } finally {
      busy = false
    }
  }

  let error = $state('')
  if (isWeb && location.hash === '#discord-signin-error') error = 'Discord 로그인에 실패했어요. 팀원으로 등록된 계정인지 확인해 주세요'

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
  <p>Discord 계정으로 로그인하면<br />나에게 배정된 업무가 한곳에 모여요.</p>
  {#if app.view.signingIn}
    <p class="wait">브라우저에서 로그인을 마치면 자동으로 돌아와요.</p>
    <button class="ghost" onclick={() => Betty.CancelSignIn()}>취소</button>
  {:else}
    <button class="discord" onclick={start}>Discord로 로그인</button>
    {#if isWeb}
      <form class="password" onsubmit={passwordLogin}>
        <span class="or">또는 이름과 비밀번호로</span>
        <input bind:value={name} placeholder="이름" autocomplete="username" aria-label="이름" />
        <input bind:value={password} type="password" placeholder="비밀번호" autocomplete="current-password" aria-label="비밀번호" />
        <button type="submit" disabled={busy || !name.trim() || !password}>로그인</button>
      </form>
    {/if}
  {/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <footer>{app.view.server?.replace(/^https?:\/\//, '')}</footer>
</section>

<style>
  .signin { height: 100%; padding-top: var(--top) !important; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 24px 28px; text-align: center; }
  .berry { width: 88px; height: 88px; margin-bottom: 2px; }
  h1 { margin: 0; font-size: 22px; letter-spacing: -0.3px; }
  p { margin: 0; color: var(--fg2); line-height: 1.5; }
  .discord { margin-top: 8px; width: 100%; max-width: 280px; height: 36px; border-radius: 10px; background: var(--discord); color: #fff; font-weight: 700; }
  .discord:hover { filter: brightness(1.08); }
  .password { width: 100%; max-width: 280px; display: flex; flex-direction: column; gap: 6px; margin-top: 10px; }
  .password .or { font-size: 11.5px; color: var(--fg3); }
  .password input { height: 34px; padding: 0 10px; border: none; outline: none; border-radius: 9px; background: var(--fill); }
  .password input:focus { box-shadow: inset 0 0 0 1.5px var(--accent); }
  .password button { height: 34px; border-radius: 9px; background: var(--fill-hover); font-weight: 700; }
  .password button:disabled { opacity: 0.5; }
  .ghost { padding: 4px 12px; border-radius: 7px; background: var(--fill); }
  .wait { font-size: 12px; }
  .error { color: var(--red); font-size: 12px; }
  footer { position: absolute; bottom: 14px; font-size: 11px; color: var(--fg3); }
</style>
