<script lang="ts">
  import { onMount } from 'svelte'
  import { app, connect } from './lib/view.svelte'
  import TaskWindow from './components/TaskWindow.svelte'
  import QuickAdd from './components/QuickAdd.svelte'

  // One bundle, two windows: "/" is the menu bar list, "/#/quick" the Spotlight-style panel.
  const route = location.hash.startsWith('#/quick') ? 'quick' : 'tasks'
  onMount(connect)
</script>

{#if app.ready}
  {#if route === 'quick'}
    <QuickAdd />
  {:else}
    <TaskWindow />
  {/if}
{/if}
