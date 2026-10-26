package tray

// labels are the tray menu texts in one language.
type labels struct {
	Open, Volume, Up, Down, Mute, Compact, CloseToTray, Quit string
}

// trayLabels returns the menu texts for lang ("en"); anything else is pt-BR.
func trayLabels(lang string) labels {
	if lang == "en" {
		return labels{"Open OPTIC", "Volume", "Up", "Down", "Mute", "Compact mode", "Close hides to tray", "Quit"}
	}
	return labels{"Abrir OPTIC", "Volume", "Aumentar", "Diminuir", "Mudo", "Modo compacto", "Fechar esconde na bandeja", "Sair"}
}
