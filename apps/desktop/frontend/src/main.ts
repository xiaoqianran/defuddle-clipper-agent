import './style.css'
import './markdown-reader.css'
import App from './App.svelte'
import { installMarkdownEnhancer } from './markdown-enhance'

const app = new App({
  target: document.getElementById('app')
})

installMarkdownEnhancer()

export default app
