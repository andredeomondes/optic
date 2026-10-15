# Android TV Remote protocol v2 — notas

Implementação em [`pkg/atvremote`](../pkg/atvremote). Mensagens protobuf codificadas à mão com `protowire` (ver [ADR 0002](adr/0002-protocolo-implementado-a-mao.md)).

## Portas

| Porta | Uso | Mensagens |
|---|---|---|
| 6467 | Pairing (Polo) | PairingRequest → Options → Configuration → Secret |
| 6466 | Controle | Configure ↔ SetActive, Ping, KeyInject, AppLink; a TV envia volume, power, app em primeiro plano e estado do teclado |

Ambas usam TLS com **certificado de cliente** autoassinado, criado na primeira execução e guardado na pasta de config (`client.crt`, `client.key`). A TV passa a confiar nele depois do pairing.

## Pairing

1. Cliente envia `PairingRequest` com o nome (`OPTIC`).
2. Negociação de `Options`/`Configuration`: codificação hexadecimal, 6 símbolos.
3. A TV mostra o código. O segredo é `SHA-256(N_cliente, e_cliente, N_tv, e_tv, código[1:])`.
4. O **1º byte do código é o checksum** do hash, então erros de digitação são detectados localmente (`ErrCodeTypo`) sem derrubar a sessão: o usuário pode tentar de novo.

## Controle

- Após o TLS, a TV manda `RemoteConfigure` (modelo, fabricante, versão). Respondemos com o nosso e com `SetActive`.
- `Ping` periódico mantém a sessão viva.
- Teclas: `KeyInject` com código Android (`KEYCODE_*`) e direção (`SHORT`, `START_LONG`, `END_LONG`). Nomes em [`keys.go`](../pkg/atvremote/keys.go).
- Apps: `AppLink` (campo 90) com um link. **A TV só aceita links `http(s)` ou esquemas com app instalado**: `market://launch?id=…`, `intent:`, `android-app://` ou um esquema sem dono voltam como `RemoteError` (campo 3, `{1: 7}`) e a TV **encerra a sessão**. Por isso o adapter (`internal/adapter/atv/applinks.go`) traduz pacotes para links validados e recusa o resto antes de enviar. Testado numa TCL (Android 9): abrem YouTube (`https://www.youtube.com`), Netflix (`https://www.netflix.com/browse`), Prime Video (`https://app.primevideo.com`), Disney+ (`https://www.disneyplus.com`) e Spotify (`spotify://`); Globoplay ainda sem link que abra o app.
- Volume: a TV **informa** o nível, mas não aceita "defina para N"; o caso de uso envia N vezes `VOLUME_UP/DOWN`.

## Teclado (IME) — observado numa TCL Google TV (Android 9)

- Ao conectar, a TV manda `20 { 1 { 12: "<pacote do app>" } }`: o app em primeiro plano.
- Quando um teclado **do sistema** abre, manda `20` de novo com o estado do campo e em seguida `21 { 1: ime_counter 2: field_counter }`. Com esses contadores, `SendText` substitui o texto do campo de uma vez.
- Os contadores só chegam quando o teclado abre: conecte o OPTIC **antes** de abrir o campo.
- Apps com teclado próprio (Netflix, Prime Video) não mandam nada; o texto não chega. A UI avisa e sugere a busca global.

## Investigando

```bash
go run ./cmd/optic watch -t 60s 192.168.0.18   # tudo que a TV envia, decodificado sem schema
go run ./cmd/optic key -v 192.168.0.18 HOME     # -v mostra cada frame
```

Campos ainda não decifrados: bitmask de features (`639`) em `RemoteConfigure`.
