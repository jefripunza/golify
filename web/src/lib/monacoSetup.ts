/**
 * monacoSetup.ts — MonacoEnvironment untuk Vite.
 *
 * Monaco 0.56 (ESM) butuh worker di-load sebagai module. Import `?worker`
 * (pola standar Vite) membuat tiap worker di-bundle jadi asset terpisah dan
 * import internal Monaco (webWorkerBootstrap.js dkk) di-rewrite otomatis —
 * menyelesaikan error:
 *   TypeError: Error resolving module specifier ".../webWorkerBootstrap.js"
 *
 * Catatan: import worker via alias `monaco-worker` → langsung ke
 * node_modules/monaco-editor/esm/vs (bypass exports map monaco yang
 * memetakan "monaco-editor/*" → "./esm/vs/*.js" dan bikin double path).
 */

import EditorWorker from 'monaco-worker/editor/editor.worker?worker'
import JsonWorker from 'monaco-worker/language/json/json.worker?worker'
import CssWorker from 'monaco-worker/language/css/css.worker?worker'
import HtmlWorker from 'monaco-worker/language/html/html.worker?worker'
import TsWorker from 'monaco-worker/language/typescript/ts.worker?worker'

let registered = false

export function ensureMonacoEnv() {
  if (registered || typeof window === 'undefined') return
  registered = true

  // @ts-expect-error — self.MonacoEnvironment di-set global
  self.MonacoEnvironment = {
    getWorker(_workerId: string, label: string) {
      switch (label) {
        case 'json':
          return new JsonWorker()
        case 'css':
        case 'scss':
        case 'less':
          return new CssWorker()
        case 'html':
        case 'handlebars':
        case 'razor':
          return new HtmlWorker()
        case 'typescript':
        case 'javascript':
          return new TsWorker()
        default:
          return new EditorWorker()
      }
    },
  }
}
