import { mount } from 'svelte'
import './lib/platform'
import './app.css'
import App from './App.svelte'

mount(App, { target: document.getElementById('app')! })
