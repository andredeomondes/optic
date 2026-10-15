# ADR 0005 — Bandeja do sistema e instância única

- **Status:** aceita
- **Data:** 2026-10-15

## Contexto

O usuário quer que fechar a janela a esconda na área de ícones ocultos do Windows (minimizar continua normal) e que abrir o app de novo traga a janela existente. Wails v2 não tem API de bandeja (só o v3, ainda alpha).

## Opções para a bandeja

| Opção | Notas |
|---|---|
| **`energye/systray`** | Fork do getlantern sem cgo no Windows, eventos de clique esquerdo/direito, roda em goroutine com thread própria |
| `getlantern/systray` | Mesma base, mais dependências, sem clique esquerdo configurável |
| Esperar Wails v3 | Alpha |

## Decisão

- Bandeja com `energye/systray` em `internal/platform/tray`, atrás de uma interface `Tray` com implementação no-op fora do Windows (no macOS a bandeja exigiria a thread principal, já usada pelo Wails).
- `OnBeforeClose` esconde a janela quando "Fechar esconde na bandeja" está ligado **e** a bandeja está pronta; caso contrário fecha. Sair de verdade: menu "Sair".
- `options.SingleInstanceLock` do Wails com `OnSecondInstanceLaunch` mostrando, restaurando e focando a janela.

## Consequências

- A janela nunca fica inacessível: sem ícone na bandeja, fechar fecha.
- Preferência salva em `prefs.json` na pasta de config (`internal/adapter/filestore`).
- O ícone da bandeja é o mesmo `build/windows/icon.ico` gerado pelo Wails a partir de `build/appicon.png`.
