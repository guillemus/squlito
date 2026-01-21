package app

import "github.com/awesome-gocui/gocui"

type loggingEditor struct {
	next gocui.Editor
}

func (editor loggingEditor) Edit(view *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	logKeyEvent("editor", key, ch, mod)
	if editor.next == nil {
		return
	}
	editor.next.Edit(view, key, ch, mod)
}
