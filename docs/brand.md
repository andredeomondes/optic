# OPTIC — identidade visual

Fonte de verdade da marca. Tokens CSS em `frontend/src/styles.css` (`:root`).

## Marca
- **Nome:** OPTIC — desktop remote/controller para Smart TV / Android TV.
- **Tagline:** “Control your TV from your PC.”
- **Personalidade:** premium, tecnológica, minimalista, rápida e confiável. Nada de dashboard SaaS genérico.

## Logo
- Símbolo: `frontend/src/assets/optic-symbol.svg` (abertura óptica/lente + tela + conexão).
- Horizontal: `frontend/src/assets/optic-horizontal.svg` quando houver espaço para símbolo + nome.
- Raster (README, redes, docs): `frontend/src/assets/raster/optic-symbol-{128,256,512}.webp`. No app, sempre SVG.
- Ícone do app: `build/appicon.png` (1024px); o `.ico` do Windows é gerado pelo Wails a partir dele.
- Respiro mínimo: 20% da largura do símbolo. Sem sombras pesadas. Em fundo escuro, logo original.
- Não redesenhar nem trocar por outro ícone.

> Os SVGs atuais são recriações vetoriais da arte oficial. Ao receber os arquivos originais, substitua-os no mesmo caminho e regenere `build/appicon.png` e os `.webp`.

## Tipografia
- Wordmark: **Space Grotesk SemiBold 600**, caixa alta, tracking ~0.12em.
- Interface: **Inter** — 400 texto, 500 labels, 600 títulos e botões.
- Fontes empacotadas via `@fontsource` (subset latin, woff2): funcionam offline.

## Cores
| Token | Valor | Uso |
|---|---|---|
| `--purple` | `#8B5CF6` | ação primária |
| `--violet` | `#6D4AFF` | apoio |
| `--blue` | `#3B82F6` | apoio |
| `--cyan` | `#22D3EE` | estados ativos/conectados |
| `--bg-solid` | `#080B18` | fundo principal |
| `--surface` | `#101426` | cards, diálogos |
| `--text` | `#F7F8FF` | texto principal |
| `--muted` | `#A8AFC3` | texto secundário |
| `--ok` | `#2EE6A6` | sucesso / TV conectada |
| `--danger` | `#FF4D6D` | power / perigo |
| `--grad` | `linear-gradient(135deg, #A855F7 0%, #6366F1 48%, #22D3EE 100%)` | identidade e controles ativos |

## Direção visual
- Dark mode obrigatório; glassmorphism sutil (blur 16px só em painéis grandes).
- Gradiente só em destaque, controles ativos e identidade.
- Bordas 1px branco em baixa opacidade; raios 14–20px.
- Sombras suaves e profundas, sem glow excessivo.
- Animações de 120 a 200ms; hover discreto, pressed evidente.
- Vermelho reservado para power/danger.
