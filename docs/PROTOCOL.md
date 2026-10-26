# Android TV Remote protocol v2 — notas

Implementação em [`pkg/atvremote`](../pkg/atvremote). Mensagens protobuf codificadas à mão com `protowire`.

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

## Roku (ECP)

HTTP simples na porta **8060**, sem pareamento nem criptografia. Endpoints usados:

| Método | Caminho | Uso |
|---|---|---|
| POST | `/keypress/<Tecla>` | `Home`, `Back`, `Select`, `Up`… `VolumeUp`, `Power` (liga/desliga), `PowerOn` |
| POST | `/keypress/Lit_<caractere>` | texto, um caractere por requisição (URL-encoded) |
| POST | `/launch/<id>` | abre o app pelo id do canal (Netflix = 12, YouTube = 837) |
| GET | `/query/apps` | apps instalados |
| GET | `/query/active-app` | app em primeiro plano |
| GET | `/query/device-info` | nome, modelo, `power-mode`, MACs |

A TV não envia eventos: o OPTIC consulta `device-info` e `active-app` a cada 3 s. **HTTP 403** significa que "Controle por apps móveis" está limitado ou desativado na TV.

## LG webOS (SSAP)

JSON por WebSocket em `wss://<ip>:3001` (certificado autoassinado; TVs antigas: `ws://<ip>:3000`).

- **Registro:** `{"type":"register","payload":{"pairingType":"PROMPT","manifest":…}}`. A TV mostra "Permitir?"; ao aceitar, responde `registered` com a `client-key`, guardada em `webos_keys.json`. Nas próximas conexões a chave vai no registro; se a TV pedir confirmação de novo ou recusar, o OPTIC pareia outra vez.
- **Requisições:** `{"type":"request","id":…,"uri":"ssap://…","payload":…}`; `subscribe` recebe atualizações com o mesmo id.

| URI | Uso |
|---|---|
| `ssap://audio/getVolume` (subscribe) | volume, máximo e mudo |
| `ssap://audio/setVolume` | volume exato |
| `ssap://system/turnOff` | desligar |
| `ssap://com.webos.service.ime/insertText` | texto no campo focado |
| `ssap://com.webos.applicationManager/listLaunchPoints` | apps instalados |
| `ssap://system.launcher/launch` | abrir app |
| `ssap://com.webos.applicationManager/getForegroundAppInfo` (subscribe) | app em primeiro plano |
| `ssap://tv/getExternalInputList`, `ssap://tv/switchInput` | entradas (HDMI…) |
| `ssap://system.notifications/createToast` | mensagem na tela |
| `ssap://com.webos.service.networkinput/getPointerInputSocket` | socket de ponteiro para teclas |

**Teclas:** pelo socket de ponteiro (outro WebSocket no mesmo host), um frame de texto por tecla: `type:button\nname:UP\n\n`.

## Samsung Tizen

- **Informações:** `GET http://<ip>:8001/api/v2/` devolve nome, modelo, `PowerState` (`on`/`standby`) e `TokenAuthSupport`.
- **Controle:** WebSocket `wss://<ip>:8002/api/v2/channels/samsung.remote.control?name=<base64 de "OPTIC">&token=<token>` (certificado autoassinado). Sem token, a TV mostra "Permitir?"; ao aceitar, envia `ms.channel.connect` com o token, guardado em `samsung_tokens.json`. Recusa: `ms.channel.unauthorized`. TVs sem `TokenAuthSupport` (2016) usam `ws://<ip>:8001` sem token.
- **Teclas:** `{"method":"ms.remote.control","params":{"Cmd":"Click","DataOfCmd":"KEY_HOME","Option":"false","TypeOfRemote":"SendRemoteKey"}}`.
- **Texto:** só com o teclado da TV aberto (eventos `ms.remote.imeStart`/`imeEnd`): `SendInputString` com o texto em base64, seguido de `SendInputEnd`.
- **Apps:** `ms.channel.emit` com `ed.installedApp.get` lista os apps; `ed.apps.launch` abre (`DEEP_LINK` para apps do tipo 2, `NATIVE_LAUNCH` para os demais).
