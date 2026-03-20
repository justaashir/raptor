package tui

import (
	"github.com/charmbracelet/huh"
)

func NewAddForm() (title, content string, err error) {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&title),
			huh.NewText().
				Title("Content (markdown)").
				Value(&content),
		),
	)
	err = form.Run()
	return
}

func NewEditForm(currentTitle, currentContent string) (title, content string, err error) {
	title = currentTitle
	content = currentContent
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&title),
			huh.NewText().
				Title("Content (markdown)").
				Value(&content),
		),
	)
	err = form.Run()
	return
}
