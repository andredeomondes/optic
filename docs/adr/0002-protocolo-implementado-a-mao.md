# ADR 0002 — Protocolo Remote v2 implementado à mão

- **Status:** aceita
- **Data:** 2026-10-14

## Contexto

O Android TV Remote protocol v2 (portas 6466/6467) usa TLS com certificado de cliente e mensagens protobuf. Os `.proto` não são publicados oficialmente; bibliotecas existentes são em Python/Node.

## Decisão

Implementar em Go puro (`pkg/atvremote`), codificando e decodificando protobuf com `google.golang.org/protobuf/encoding/protowire`, sem `.proto` nem geração de código. Um dumper sem schema (`DumpRaw`) mostra qualquer mensagem, o que viabiliza a engenharia reversa via `optic watch`.

## Consequências

- Zero dependências de runtime além da stdlib + `protowire`.
- Campos desconhecidos são ignorados com segurança; novos campos são lidos por número.
- Precisamos de testes de ida e volta (`pairing_test.go`, `ime_test.go`) para não regredir o formato.
