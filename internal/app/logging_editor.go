package app

import "github.com/awesome-gocui/gocui"

type loggingEditor struct {
	next gocui.Editor
	app  *App
}

func (editor loggingEditor) Edit(view *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	logKeyEvent("editor", key, ch, mod)
	if editor.next == nil {
		return
	}
	editor.next.Edit(view, key, ch, mod)
	if editor.app == nil {
		return
	}
	if isQueryEditKey(key, ch, mod) {
		editor.app.resetHistorySelection()
	}
}

func isQueryEditKey(key gocui.Key, ch rune, mod gocui.Modifier) bool {
	if ch != 0 {
		return true
	}

	switch key {
	case gocui.KeySpace:
		return true
	case gocui.KeyBackspace:
		return true
	case gocui.KeyBackspace2:
		return true
	case gocui.KeyDelete:
		return true
	case gocui.KeyTab:
		return true
	case gocui.KeyEnter:
		return true
	default:
		return false
	}
}
