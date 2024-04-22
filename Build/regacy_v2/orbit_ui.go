package main

import (
	"os/exec"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/ini.v1"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Orbit")
	myWindow.Resize(fyne.NewSize(300, 320)) // ウィンドウのサイズを設定

	// 画像を読み込む
	banner := canvas.NewImageFromFile("../Img/banner.png")
	banner.FillMode = canvas.ImageFillOriginal // 画像のサイズを変更せずに表示

	cfg, err := ini.Load("config.ini")
	if err != nil {
		dialog.ShowError(err, myWindow)
		return
	}

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
		appPath := cfg.Section("").Key(app).String()
		launchApplication(project, appPath, version, myWindow)
	})

	menuBar := fyne.NewMainMenu(
		fyne.NewMenu("File",
			fyne.NewMenuItem("Edit Config", func() { showConfigEditor(myApp, cfg) }),
		),
	)
	myWindow.SetMainMenu(menuBar)

	// ウィジェットコンテンツの作成
	content := container.NewVBox(
		banner, // ここで画像を追加
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

func launchApplication(project, appPath, version string, window fyne.Window) {
	cmd := exec.Command(appPath, "--version", version)
	output, err := cmd.CombinedOutput()
	if err != nil {
		dialog.ShowError(err, window)
	} else {
		dialog.ShowInformation("Launch Success", "Output: "+string(output), window)
	}
}

func showConfigEditor(app fyne.App, cfg *ini.File) {
	w := app.NewWindow("Edit Config") // 新しいウィンドウを作成
	w.Resize(fyne.NewSize(665, 275))  // ウィンドウのサイズを設定

	form := &widget.Form{}
	// 各アプリケーション名と対応するパスをテキストボックスに事前に表示
	for _, app := range []string{"Maya", "Blender", "AfterEffects", "Photoshop"} {
		entry := widget.NewEntry()
		entry.SetText(cfg.Section("").Key(app).String()) // config.ini からパスを読み込み、テキストボックスに設定
		form.Append(app, entry)
	}

	saveButton := widget.NewButton("Save", func() {
		// フォームの各エントリから新しい値を取得して設定ファイルを更新
		for i, app := range []string{"Maya", "Blender", "AfterEffects", "Photoshop"} {
			cfg.Section("").Key(app).SetValue(form.Items[i].Widget.(*widget.Entry).Text)
		}
		// 設定をファイルに保存
		if err := cfg.SaveTo("config.ini"); err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Config Saved", "Configuration has been saved successfully.", w)
			w.Close()
		}
	})

	cancelButton := widget.NewButton("Cancel", func() {
		w.Close()
	})

	content := container.NewVBox(
		form,
		container.NewHBox(saveButton, cancelButton),
	)

	w.SetContent(content)
	w.Show()
}
