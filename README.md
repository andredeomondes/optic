<div align="center">

<img src="frontend/src/assets/raster/optic-symbol-256.webp" width="112" alt="OPTIC" />

# OPTIC

**Controle sua TV pelo computador.** Sem hardware extra, sem nuvem, sem conta.

Controle remoto desktop open source para **Android TV e Google TV**, falando direto com a TV pela rede local com o Android TV Remote protocol v2, implementado do zero em Go.

[**Baixar**](https://github.com/andredeomondes/optic/releases/latest) ·
[**Guia de instalação e uso**](https://andredeomondes.github.io/optic/) ·
[Arquitetura](docs/ARCHITECTURE.md) ·
[Protocolo](docs/PROTOCOL.md)

![CI](https://github.com/andredeomondes/optic/actions/workflows/ci.yml/badge.svg)
![Licença MIT](https://img.shields.io/badge/licen%C3%A7a-MIT-8B5CF6)
![Windows · macOS · Linux](https://img.shields.io/badge/Windows%20%C2%B7%20macOS%20%C2%B7%20Linux-22D3EE)

<img src="docs/screenshots/main.png" alt="Janela principal do OPTIC conectada a uma TV" width="820" />

</div>

## Recursos

- **Encontra a TV sozinho** na rede (mDNS) e pareia com o código de 6 caracteres que aparece na tela.
- **Liga a TV** pela rede (Wake-on-LAN), depois da primeira conexão com ela ligada.
- **Controle completo:** D-pad, voltar, início, volume (com slider), canais, mídia e power.
- **Abre apps** com um clique: YouTube, Netflix, Prime Video, Globoplay e qualquer outro pelo nome do pacote.
- **Digita na TV** pelo teclado do PC, direto no campo de busca. `Ctrl+V` cola o texto copiado na TV.
- **Comandos rápidos** (`Ctrl+K`): "netflix", "volume 20", "digitar …" e outros.
- **Atalhos de teclado:** setas navegam, `Enter` OK, `Esc` voltar, `H` início, `M` mudo, `+`/`-` volume, espaço play/pause, `Ctrl+T` digitar.
- **Volume pela roda do mouse** sobre o controle (com a TV conectada) e pelo menu da bandeja.
- **Timer para desligar** a TV em 15, 30, 60 ou 90 minutos (ou "desligar em 45 min" no `Ctrl+K`).
- **Aviso de nova versão:** o app avisa quando sai uma release nova (dá para desligar em Configurações). A checagem só consulta a página pública de releases no GitHub; nenhum dado seu ou da TV é enviado.
- **Modo compacto** 360×520 que pode ficar sempre no topo, ao lado do que você estiver fazendo.
- **Bandeja do sistema** (Windows): fechar esconde o app na área de ícones ocultos; abrir de novo traz a mesma janela.
- **Tela cheia** com `F11` e janela maximizável.
- **100% local:** o controle não passa por servidores. O certificado do pareamento fica só no seu PC.

> Para ligar a TV pela rede, a opção **Ligar pela rede** (ou *Network standby*) precisa estar ativa na TV. Samsung, LG e Roku já aparecem na lista de TVs; o suporte a elas chega nas próximas versões.

<table>
  <tr>
    <td align="center"><img src="docs/screenshots/compact.png" width="250" alt="Modo compacto" /><br/><sub>Modo compacto</sub></td>
    <td align="center"><img src="docs/screenshots/quick-command.png" width="420" alt="Comandos rápidos" /><br/><sub>Comandos rápidos (Ctrl+K)</sub></td>
  </tr>
  <tr>
    <td colspan="2" align="center"><img src="docs/screenshots/settings.png" width="420" alt="Configurações" /><br/><sub>Configurações › Janela</sub></td>
  </tr>
</table>

## Instalar

Baixe em [**Releases**](https://github.com/andredeomondes/optic/releases/latest):

| Sistema | Arquivo | Como instalar |
|---|---|---|
| Windows 10/11 | `OPTIC-windows-amd64-setup.exe` | Abra e siga o instalador. Atalho no menu Iniciar e na área de trabalho. |
| macOS 11+ (Intel e Apple Silicon) | `OPTIC-macos-universal.dmg` | Arraste o OPTIC para Aplicativos. Na 1ª vez: botão direito › **Abrir**. |
| Linux (x64) | `OPTIC-linux-amd64.tar.gz` | Extraia e rode `./install.sh` (instala no seu usuário, sem root). |

O passo a passo, incluindo os avisos do SmartScreen e do Gatekeeper (os binários ainda não são assinados), está no [**guia**](https://andredeomondes.github.io/optic/).

## Primeiro uso

1. PC e TV na **mesma rede** (Wi-Fi ou cabo).
2. Abra o OPTIC: a TV aparece em **TVs na rede** (ou use **Adicionar TV** com o IP).
3. Clique nela. A TV mostra um **código de 6 caracteres**: digite no OPTIC.
4. Pronto. Das próximas vezes o OPTIC reconecta sozinho.

> Netflix e Prime Video usam teclado próprio, que não recebe texto pelo protocolo. Use as setas e o OK, ou a busca global da TV.

## Desenvolvimento

```bash
git clone https://github.com/andredeomondes/optic && cd optic
wails build -skipbindings                  # app em build/bin/
wails build -skipbindings -nsis            # + instalador Windows
cd frontend && npm install && npm run dev  # só a interface, com backend falso
```

Testes: `go test ./...` e, no frontend, `npx tsc --noEmit && npm test`. Detalhes em [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

### Estrutura

```
main.go · app.go · shell.go   composição + entrega via Wails
internal/domain               tipos, erros e portas
internal/control              casos de uso (testados com fakes)
internal/adapter/             atv (protocolo) · mdns (descoberta) · filestore (config)
internal/platform/tray        bandeja do sistema
pkg/atvremote                 Android TV Remote v2: pairing (6467) e controle (6466)
pkg/discovery                 mDNS
cmd/optic                     CLI de diagnóstico (discover, pair, key, type, watch)
frontend/src                  lib · hooks · components
site/                         guia de instalação e uso (GitHub Pages)
```

Mais detalhes em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

### CLI

```bash
go run ./cmd/optic discover -v
go run ./cmd/optic pair 192.168.0.18
go run ./cmd/optic key 192.168.0.18 HOME DPAD_DOWN DPAD_CENTER
go run ./cmd/optic watch 192.168.0.18    # tudo que a TV envia, decodificado
```

## Roadmap

- [ ] Prévia da tela da TV via ADB
- [ ] Digitação em apps com teclado próprio (ADB `input text`)
- [ ] Binários assinados (Windows e macOS)
- [ ] Bandeja no macOS e Linux

## Licença

[MIT](LICENSE)
