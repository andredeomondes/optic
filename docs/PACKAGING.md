# Empacotamento e distribuição (guia do mantenedor)

**English summary.** Every release tag publishes `SHA256SUMS` and `packaging-manifests.zip`. The zip holds ready-to-submit manifests for winget, scoop, Homebrew, AUR and Flatpak, rendered by `go run ./cmd/pkgmeta`. Nothing is published automatically: submitting to each registry needs the maintainer's own accounts (GitHub for winget/scoop/Homebrew, an AUR account with an SSH key). AUR and Flatpak use the WebKitGTK 4.1 Linux tarball (`OPTIC-linux-amd64-webkit41.tar.gz`); both are still "coming soon", and Flathub may ask for a build from source. Binaries are unsigned (SmartScreen / Gatekeeper warnings). Nothing described here is published yet; the install commands in the READMEs work only after each submission is accepted.

Este guia explica como levar os manifestos de cada release para os gerenciadores de pacotes. **Nada disso está publicado ainda**: os comandos de instalação dos READMEs só passam a funcionar depois que cada envio for feito e aceito.

## O que cada release gera

O workflow de release (`.github/workflows/release.yml`) roda a cada tag e publica, além dos instaladores:

- `SHA256SUMS`: checksums dos instaladores.
- `packaging-manifests.zip`: manifestos prontos, gerados por `cmd/pkgmeta` a partir da versão e do `SHA256SUMS`.

Conteúdo do zip:

| Pasta | Arquivos | Destino |
|---|---|---|
| `winget/` | `andredeomondes.OPTIC.yaml` (version), `.installer.yaml`, `.locale.en-US.yaml`, `.locale.pt-BR.yaml` | `microsoft/winget-pkgs` |
| `scoop/` | `optic.json` | bucket próprio `scoop-optic` |
| `homebrew/` | `optic.rb` (cask) | tap próprio `homebrew-optic` |
| `aur/` | `PKGBUILD`, `.SRCINFO` (pacote `optic-bin`) | AUR |
| `flatpak/` | `io.github.andredeomondes.optic.yml` e `.metainfo.xml` | Flathub (veja abaixo) |

## Regerar os manifestos localmente

Baixe o `SHA256SUMS` da release e rode, na raiz do repositório:

```sh
go run ./cmd/pkgmeta -version vX.Y.Z -sums SHA256SUMS -out out/
```

Os modelos ficam em `internal/pkgmeta/templates`. Para alterar um manifesto, edite o modelo, rode `go test ./internal/pkgmeta ./cmd/pkgmeta` e gere de novo.

## Binários sem assinatura

Os instaladores ainda não são assinados:

- **Windows**: o SmartScreen mostra "O Windows protegeu seu computador". Clique em **Mais informações › Executar assim mesmo**. O winget e o scoop não evitam o aviso do SmartScreen na primeira execução.
- **macOS**: o Gatekeeper bloqueia o app baixado. Depois de instalar (DMG ou Homebrew), rode `xattr -dr com.apple.quarantine /Applications/OPTIC.app`; ou, no macOS 15 ou mais novo, abra o app uma vez e vá em **Ajustes do Sistema › Privacidade e Segurança › Abrir Mesmo Assim**. O cask exige macOS 11 (Big Sur) ou mais novo.

## winget

Pacote: `andredeomondes.OPTIC`. Instalador `nullsoft` (NSIS), escopo `machine`, localidades en-US e pt-BR.

Precisa de conta GitHub do mantenedor. Duas formas:

1. Pull request manual no repositório `microsoft/winget-pkgs`: faça fork, copie os quatro arquivos de `winget/` para `manifests/a/andredeomondes/OPTIC/<versão>/` (sem o "v" na versão) e abra o PR. O pipeline de validação do winget roda no PR e pode pedir ajustes.
2. Com o `wingetcreate`: `wingetcreate submit` apontando para a pasta com os manifestos (exige um token GitHub pessoal do mantenedor).

Teste local antes de enviar: `winget validate <pasta>` e `winget install --manifest <pasta>` (a instalação com manifesto local exige habilitar `winget settings --enable LocalManifestFiles`, uma configuração do próprio usuário).

## Scoop

Manifesto portátil (`optic.json`, o `.exe` portátil); o `autoupdate` usa o `SHA256SUMS` da release.

Precisa de conta GitHub do mantenedor. Crie o repositório `andredeomondes/scoop-optic` e coloque o arquivo em `bucket/optic.json`:

```sh
git clone https://github.com/andredeomondes/scoop-optic
mkdir -p scoop-optic/bucket && cp out/scoop/optic.json scoop-optic/bucket/
cd scoop-optic && git add . && git commit -m "optic vX.Y.Z" && git push
```

Depois disso: `scoop bucket add optic https://github.com/andredeomondes/scoop-optic` e `scoop install optic`.

## Homebrew

Cask (`optic.rb`) que instala o `.dmg` universal, sem assinatura.

Precisa de conta GitHub do mantenedor. Crie o repositório `andredeomondes/homebrew-optic` e coloque o arquivo em `Casks/optic.rb`:

```sh
git clone https://github.com/andredeomondes/homebrew-optic
mkdir -p homebrew-optic/Casks && cp out/homebrew/optic.rb homebrew-optic/Casks/
cd homebrew-optic && git add . && git commit -m "optic vX.Y.Z" && git push
```

Instalação pelo usuário: `brew install --cask andredeomondes/optic/optic`. O Homebrew 5 removeu o `--no-quarantine`; veja o `xattr` acima por causa do Gatekeeper.

## AUR

Pacote `optic-bin` (instala o binário do tarball WebKitGTK 4.1, a `LICENSE` e o ícone de 512 px; depende de `webkit2gtk-4.1`, pois o Arch abandonou o `webkit2gtk` 4.0). Em breve: ainda não publicado no AUR.

Precisa de conta no AUR com chave SSH cadastrada. Na primeira vez o clone vem vazio:

```sh
git clone ssh://aur@aur.archlinux.org/optic-bin.git
cp out/aur/PKGBUILD out/aur/.SRCINFO optic-bin/
cd optic-bin
makepkg --printsrcinfo > .SRCINFO   # confira que não mudou nada em relação ao gerado
git add PKGBUILD .SRCINFO && git commit -m "optic vX.Y.Z" && git push
```

Teste antes: `makepkg -si` num Arch (ou contêiner Arch).

## Flatpak / Flathub

O manifesto (`flatpak/`) usa o tarball WebKitGTK 4.1 da release (`OPTIC-linux-amd64-webkit41.tar.gz`, compilado com Wails `-tags webkit2_41`), o runtime GNOME 50 e o ícone de 512 px. Em breve: ainda não foi enviado ao Flathub.

1. Testar com `flatpak-builder --user --install build io.github.andredeomondes.optic.yml`.
2. Ao enviar, use o runtime GNOME vigente na época (atualize `runtime-version`, hoje `'50'`).
3. Enviar para o Flathub: PR no repositório `flathub/flathub` (branch `new-pr`) seguindo o guia de submissão deles, com conta GitHub do mantenedor; ou manter um repositório Flatpak próprio.
4. O Flathub pode pedir compilação a partir do código-fonte em vez do binário pré-compilado; nesse caso o manifesto precisa de um módulo que rode o build do Wails.

Enquanto isso, não anuncie Flatpak como disponível.

## Checklist por release

1. Tag `vX.Y.Z` publicada; o workflow termina e anexa `SHA256SUMS` e `packaging-manifests.zip`.
2. Baixar o zip (ou regerar localmente, veja acima).
3. Enviar winget (PR), atualizar os repositórios `scoop-optic` e `homebrew-optic`, atualizar o AUR.
4. Conferir as instalações em máquinas limpas.
