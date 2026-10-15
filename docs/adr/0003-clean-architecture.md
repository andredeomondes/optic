# ADR 0003 — Núcleo em camadas (portas e adapters)

- **Status:** aceita
- **Data:** 2026-10-15

## Contexto

O primeiro `app.go` misturava bindings Wails, regras (volume em passos, pairing com retry, reconexão), acesso a disco, eventos de UI e bandeja. Era impossível testar sem uma TV e difícil suportar outra marca de TV.

## Decisão

Separar em camadas com a regra de dependência apontando para dentro:

- `internal/domain`: tipos, erros e portas (`Connector`, `Session`, `Pairer`, `Discoverer`, `HostStore`, `EventSink`).
- `internal/control`: casos de uso, dependendo só das portas, injetadas por `control.Deps`.
- `internal/adapter/{atv,mdns,filestore}` e `internal/platform/tray`: implementações reais.
- `main`: composition root e entrega via Wails (`App` só delega; `Shell` cuida de janela e bandeja).

## Consequências

- Casos de uso testados com fakes (`internal/control/service_test.go`), sem rede.
- Os nomes dos bindings JS não mudaram (`App` continua no pacote `main`), então o frontend não foi tocado.
- Suportar outro protocolo = novo adapter + uma linha em `main.go`.
- Um pouco mais de arquivos; cada um com um papel só.
