# ADR 0004 — Bindings Wails mantidos à mão

- **Status:** aceita
- **Data:** 2026-10-15

## Contexto

`wails build` gera `frontend/wailsjs/go/main/App.{js,d.ts}` compilando e executando um exe temporário. Políticas de Application Control / Smart App Control do Windows bloqueiam esse exe, quebrando o build.

## Decisão

Compilar com `wails build -skipbindings` e manter `App.js` e `App.d.ts` à mão. Todo acesso ao Go passa por `frontend/src/lib/backend.ts`, e `frontend/src/devStub.ts` simula os mesmos métodos no navegador.

## Consequências

- Ao adicionar/renomear método exportado em `App`/`Shell`, atualizar: `App.js`, `App.d.ts`, `lib/backend.ts`, `devStub.ts`.
- `npx tsc --noEmit` pega divergências de tipo no frontend.
- No CI (sem essas políticas) o build também usa `-skipbindings`, para ser idêntico ao local.
