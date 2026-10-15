# Desenvolvimento

## Requisitos

| Ferramenta | Versão |
|---|---|
| Go | 1.26+ |
| Node.js | 20+ |
| Wails CLI | v2.12 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`) |
| NSIS (opcional, instalador Windows) | 3.x (`winget install NSIS.NSIS`) |

Linux também precisa de `libgtk-3-dev` e `libwebkit2gtk-4.1-dev` (ver `wails doctor`).

## Comandos

```bash
# app desktop
wails build -skipbindings                 # build/bin/optic-desktop(.exe)
wails build -skipbindings -nsis           # + instalador Windows (NSIS)
wails dev -skipbindings                   # hot reload

# frontend sozinho, com backend falso (src/devStub.ts)
cd frontend && npm install && npm run dev

# qualidade
go vet ./... && go test ./...
cd frontend && npx tsc --noEmit && npm test

# CLI
go run ./cmd/optic discover -v
```

## Fluxo de mudança

1. Regra de negócio nova → `internal/control` com teste usando fakes.
2. Integração nova (protocolo, disco, SO) → adapter em `internal/adapter` ou `internal/platform`, implementando uma porta de `internal/domain`.
3. Exposta ao frontend → método em `App`/`Shell` e atualização manual dos bindings ([ADR 0004](adr/0004-bindings-mantidos-a-mao.md)): `frontend/wailsjs/go/main/App.{js,d.ts}`, `frontend/src/lib/backend.ts`, `frontend/src/devStub.ts`.
4. UI → componente em `frontend/src/components`, estado em `frontend/src/hooks`.

## Commits

[Conventional Commits](https://www.conventionalcommits.org), pequenos e com um propósito cada:

```
feat(tray): esconder janela ao fechar
fix(window): aplicar tamanho mínimo antes do tamanho
refactor(control): extrair casos de uso do app.go
docs(adr): registrar escolha do systray
build(ci): gerar dmg no macos-latest
```

## Releases

Uma tag `vX.Y.Z` dispara [`.github/workflows/release.yml`](../.github/workflows/release.yml), que compila e anexa ao GitHub Release:

- Windows: instalador NSIS (`OPTIC-amd64-installer.exe`) e exe portátil.
- macOS: `OPTIC.dmg` universal (Intel + Apple Silicon).
- Linux: binário `optic-desktop` (`.tar.gz`).

Os binários **não são assinados**. No Windows o SmartScreen/Smart App Control pode avisar ou bloquear; no macOS o Gatekeeper pede "Abrir" pelo menu de contexto na primeira vez. O site explica os dois casos.

## Problemas conhecidos

- **"Uma política de Controle de Aplicativo bloqueou este arquivo"** (Windows): Smart App Control avalia cada exe novo sem assinatura e às vezes bloqueia um build e libera o seguinte. Recompile mudando o binário (`wails build -skipbindings -trimpath`). Solução definitiva: assinar com certificado confiável.
- **Ícone antigo na barra de tarefas** depois de trocar `build/appicon.png`: cache de ícones do Explorer. `ie4uinit.exe -show` ou reiniciar o Explorer.
