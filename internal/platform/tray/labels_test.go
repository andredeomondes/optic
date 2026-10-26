package tray

import "testing"

func TestTrayLabels(t *testing.T) {
	pt := trayLabels("pt-BR")
	if pt.Open != "Abrir OPTIC" || pt.Up != "Aumentar" || pt.Down != "Diminuir" || pt.Mute != "Mudo" ||
		pt.Compact != "Modo compacto" || pt.CloseToTray != "Fechar esconde na bandeja" || pt.Quit != "Sair" || pt.Volume != "Volume" {
		t.Fatalf("pt-BR: %+v", pt)
	}
	en := trayLabels("en")
	want := labels{"Open OPTIC", "Volume", "Up", "Down", "Mute", "Compact mode", "Close hides to tray", "Quit"}
	if en != want {
		t.Fatalf("en: %+v", en)
	}
	for _, l := range []string{"", "fr", "EN", "pt"} {
		if trayLabels(l) != pt {
			t.Errorf("lang %q must fall back to pt-BR", l)
		}
	}
}
