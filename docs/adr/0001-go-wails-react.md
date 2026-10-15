# ADR 0001 — Go + Wails v2 + React

- **Status:** aceita
- **Data:** 2026-10-14

## Contexto

O protocolo da TV (TLS + protobuf), o mDNS e a CLI já estavam em Go. Faltava uma interface desktop bonita, leve e multiplataforma (Windows primeiro, macOS e Linux depois).

## Opções

| Opção | Prós | Contras |
|---|---|---|
| **Wails v2** | Backend Go nativo (reuso total), WebView do sistema, exe ~10 MB, instalador NSIS pronto | Sem bandeja nativa no v2; bindings gerados por exe temporário |
| Tauri | Leve, maduro | Backend em Rust: reescrever protocolo ou rodar Go como sidecar |
| Electron | Ecossistema enorme | ~150 MB, Node no meio do caminho |
| Fyne / Gio | Go puro | Visual glassmorphism muito mais trabalhoso |

## Decisão

Wails v2 com frontend React + TypeScript + Vite.

## Consequências

- O código Go é chamado direto do JS (`window.go.main.App.*`), sem servidor HTTP local.
- Precisamos de soluções próprias para bandeja (ADR 0005) e para a geração de bindings (ADR 0004).
- Builds de macOS precisam rodar num Mac (feito no CI, `macos-latest`).
