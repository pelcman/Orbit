package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Orbit")

	projectInput := widget.NewEntry()
	projectInput.SetPlaceHolder("Enter Project Name")

	appSelect := widget.NewSelect([]string{"Maya", "Blender", "AfterEffects", "Photoshop"}, func(value string) {})
	appSelect.SetSelected("Maya") // Default selection

	versionInput := widget.NewEntry()
	versionInput.SetPlaceHolder("Enter Version")

	launchButton := widget.NewButton("Launch", func() {
		project := projectInput.Text
		app := appSelect.Selected
		version := versionInput.Text
		launchApplication(project, app, version, myWindow)
	})

	content := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Set Project", projectInput),
			widget.NewFormItem("Application", appSelect),
			widget.NewFormItem("Use Version", versionInput),
		),
		launchButton,
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func launchApplication(project, app, version string, window fyne.Window) {
	// Update this path to the actual application path
	cmd := exec.Command("path/to/"+app, "--version", version)
	output, err := cmd.CombinedOutput() // Capturing output and error
	if err != nil {
		dialog.ShowError(err, window)
	} else {
		dialog.ShowInformation("Launch Success", "Output: "+string(output), window)
	}
}
