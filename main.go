package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	githubAPIURL     = "https://api.github.com/repos/comfyanonymous/ComfyUI/releases"
	packageDir       = "packages"
	tempDir          = "temp"
	logsDir          = "Logs"
	configFile       = "orbit_config.json"
	downloadFileName = "ComfyUI_windows_portable_nvidia.7z"
)

var (
	logger  *log.Logger
	logFile *os.File
)

type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type CustomApp struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Icon string `json:"icon"`
}

type Config struct {
	LastVersion string      `json:"last_version"`
	CustomApps  []CustomApp `json:"custom_apps"`
	GPUType     string      `json:"gpu_type"` // "nvidia", "amd", "cpu"
}

type OrbitApp struct {
	app             fyne.App
	window          fyne.Window
	releases        []Release
	config          Config
	statusLabel     *widget.Label
	versionSelect   *widget.Select
	installedLabel  *widget.Label
	launchButton    *widget.Button
	gpuSelect       *widget.Select
	selectedVersion string
}

func main() {
	// ロギングを初期化
	initLogger()
	defer closeLogger()

	logger.Println("=== Orbit Started ===")
	logger.Printf("OS: %s, Arch: %s\n", runtime.GOOS, runtime.GOARCH)

	orbitApp := &OrbitApp{}
	orbitApp.app = app.New()
	orbitApp.window = orbitApp.app.NewWindow("Orbit")
	orbitApp.window.Resize(fyne.NewSize(200, 200))
	orbitApp.window.SetFixedSize(false)

	logger.Println("Loading configuration...")
	orbitApp.loadConfig()

	logger.Println("Setting up UI...")
	orbitApp.setupModernUI()

	logger.Println("Showing main window...")
	orbitApp.window.ShowAndRun()

	logger.Println("=== Orbit Closed ===")
}

// ロガーを初期化
func initLogger() {
	// Logsディレクトリを作成
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		return
	}

	// ログファイル名（タイムスタンプ付き）
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("orbit_%s.log", timestamp))

	// ログファイルを開く
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		logger = log.New(os.Stdout, "[Orbit] ", log.LstdFlags|log.Lshortfile)
		return
	}

	// マルチライター（ファイルと標準出力の両方に出力）
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger = log.New(multiWriter, "[Orbit] ", log.LstdFlags|log.Lshortfile)

	logger.Printf("Log file created: %s\n", logPath)
}

// ロガーを閉じる
func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

func (o *OrbitApp) loadConfig() {
	data, err := os.ReadFile(configFile)
	if err == nil {
		json.Unmarshal(data, &o.config)
	}

	// デフォルトのカスタムアプリを設定（空の場合）
	if len(o.config.CustomApps) == 0 {
		o.config.CustomApps = []CustomApp{
			{Name: "App 1", Path: "", Icon: ""},
			{Name: "App 2", Path: "", Icon: ""},
			{Name: "App 3", Path: "", Icon: ""},
			{Name: "App 4", Path: "", Icon: ""},
			{Name: "App 5", Path: "", Icon: ""},
			{Name: "App 6", Path: "", Icon: ""},
		}
	}

	// GPUタイプのデフォルト設定（自動検出）
	if o.config.GPUType == "" {
		o.config.GPUType = o.detectGPU()
		o.saveConfig()
	}
}

// GPU検出機能
func (o *OrbitApp) detectGPU() string {
	// nvidia-smiコマンドでNVIDIA GPUを検出
	cmd := exec.Command("nvidia-smi")
	if err := cmd.Run(); err == nil {
		return "nvidia"
	}

	// dxdiagやwmicでAMD GPUを検出
	cmd = exec.Command("wmic", "path", "win32_VideoController", "get", "name")
	output, err := cmd.Output()
	if err == nil {
		outputStr := strings.ToLower(string(output))
		if strings.Contains(outputStr, "amd") || strings.Contains(outputStr, "radeon") {
			return "amd"
		}
		if strings.Contains(outputStr, "nvidia") || strings.Contains(outputStr, "geforce") || strings.Contains(outputStr, "rtx") {
			return "nvidia"
		}
	}

	// デフォルトはCPU
	return "cpu"
}

func (o *OrbitApp) saveConfig() {
	data, _ := json.MarshalIndent(o.config, "", "  ")
	os.WriteFile(configFile, data, 0644)
}

func (o *OrbitApp) setupModernUI() {
	// === 上部: ロゴ ===
	banner := canvas.NewImageFromFile("Img/banner.png")
	banner.FillMode = canvas.ImageFillContain
	banner.SetMinSize(fyne.NewSize(400, 120))

	// === 左下: バージョン選択とインストール状態 ===
	o.versionSelect = widget.NewSelect([]string{"Loading..."}, func(value string) {
		o.selectedVersion = value
		o.updateInstalledStatus()
	})
	o.versionSelect.PlaceHolder = "Select ComfyUI Version"

	o.installedLabel = widget.NewLabel("Not Installed")
	o.installedLabel.TextStyle = fyne.TextStyle{Bold: true}

	versionCard := container.NewVBox(
		widget.NewLabel("ComfyUI Version:"),
		o.versionSelect,
		container.NewHBox(
			canvas.NewCircle(theme.ErrorColor()),
			o.installedLabel,
		),
	)

	// === 右下: GPU選択オプション ===
	o.gpuSelect = widget.NewSelect([]string{"NVIDIA GPU", "AMD GPU", "CPU"}, func(value string) {
		// 選択に応じてGPUタイプを保存
		switch value {
		case "NVIDIA GPU":
			o.config.GPUType = "nvidia"
		case "AMD GPU":
			o.config.GPUType = "amd"
		case "CPU":
			o.config.GPUType = "cpu"
		}
		o.saveConfig()
		o.updateInstalledStatus() // 異なるGPUタイプは異なるパッケージなので再確認
	})

	// 現在の設定に応じて選択
	switch o.config.GPUType {
	case "nvidia":
		o.gpuSelect.SetSelected("NVIDIA GPU")
	case "amd":
		o.gpuSelect.SetSelected("AMD GPU")
	case "cpu":
		o.gpuSelect.SetSelected("CPU")
	}

	detectedGPU := o.detectGPU()
	detectedLabel := widget.NewLabel(fmt.Sprintf("Detected: %s", strings.ToUpper(detectedGPU)))
	detectedLabel.TextStyle = fyne.TextStyle{Italic: true}

	optionsCard := container.NewVBox(
		widget.NewLabel("GPU Type:"),
		o.gpuSelect,
		detectedLabel,
	)

	// 左右のカードを横並び
	middleSection := container.NewGridWithColumns(2,
		versionCard,
		optionsCard,
	)

	// === カスタムアプリアイコン（6個） ===
	appIconsContainer := o.createCustomAppIcons()

	// === 下部: Launchボタン ===
	o.launchButton = widget.NewButton("Launch ComfyUI", func() {
		o.updateStatus("Launch button clicked!")
		if o.selectedVersion == "" {
			dialog.ShowError(fmt.Errorf("Please select a version"), o.window)
			return
		}
		o.updateStatus(fmt.Sprintf("Selected version: %s", o.selectedVersion))
		o.launchComfyUI()
	})
	o.launchButton.Importance = widget.HighImportance
	o.launchButton.Disable()

	// リフレッシュボタン
	refreshButton := widget.NewButton("Refresh Versions", func() {
		go o.fetchReleases()
	})

	buttonRow := container.NewHBox(
		layout.NewSpacer(),
		refreshButton,
		o.launchButton,
		layout.NewSpacer(),
	)

	// === ステータスバー ===
	o.statusLabel = widget.NewLabel("Click 'Refresh Versions' to load ComfyUI releases")
	o.statusLabel.Wrapping = fyne.TextWrapWord
	o.statusLabel.TextStyle = fyne.TextStyle{Italic: true}

	// === メインレイアウト ===
	content := container.NewBorder(
		// Top
		container.NewVBox(
			banner,
			widget.NewSeparator(),
		),
		// Bottom
		container.NewVBox(
			widget.NewSeparator(),
			appIconsContainer,
			buttonRow,
			widget.NewSeparator(),
			o.statusLabel,
		),
		// Left & Right
		nil, nil,
		// Center
		container.NewVBox(
			widget.NewLabel(""),
			middleSection,
		),
	)

	o.window.SetContent(content)

	// 自動的にリリースを取得
	go o.fetchReleases()
}

func (o *OrbitApp) createCustomAppIcons() *fyne.Container {
	icons := container.NewHBox()

	for i := 0; i < 6; i++ {
		idx := i
		appBtn := widget.NewButton(o.config.CustomApps[i].Name, func() {
			o.launchCustomApp(idx)
		})
		appBtn.Icon = theme.DocumentIcon()

		// 設定ボタン（右クリック代わり）
		settingsBtn := widget.NewButton("⚙", func() {
			o.showCustomAppSettings(idx)
		})
		settingsBtn.Importance = widget.LowImportance

		appContainer := container.NewVBox(
			appBtn,
			settingsBtn,
		)

		icons.Add(appContainer)
	}

	return container.NewVBox(
		widget.NewLabel("Custom Apps:"),
		icons,
	)
}

func (o *OrbitApp) showCustomAppSettings(index int) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(o.config.CustomApps[index].Name)

	pathEntry := widget.NewEntry()
	pathEntry.SetText(o.config.CustomApps[index].Path)

	browseButton := widget.NewButton("Browse...", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				pathEntry.SetText(reader.URI().Path())
				reader.Close()
			}
		}, o.window)
	})

	form := container.NewVBox(
		widget.NewLabel("App Name:"),
		nameEntry,
		widget.NewLabel("App Path:"),
		pathEntry,
		browseButton,
	)

	dialog.ShowCustomConfirm("Configure Custom App", "Save", "Cancel", form, func(save bool) {
		if save {
			o.config.CustomApps[index].Name = nameEntry.Text
			o.config.CustomApps[index].Path = pathEntry.Text
			o.saveConfig()
			o.setupModernUI() // UIを再構築
		}
	}, o.window)
}

func (o *OrbitApp) launchCustomApp(index int) {
	app := o.config.CustomApps[index]
	if app.Path == "" {
		dialog.ShowInformation("Not Configured", fmt.Sprintf("%s is not configured yet.\nClick the ⚙ button to set up.", app.Name), o.window)
		return
	}

	cmd := exec.Command(app.Path)
	if err := cmd.Start(); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to launch %s: %v", app.Name, err), o.window)
	}
}

func (o *OrbitApp) updateInstalledStatus() {
	if o.selectedVersion == "" {
		logger.Println("updateInstalledStatus: No version selected")
		o.installedLabel.SetText("✗ Not Installed")
		o.installedLabel.Importance = widget.WarningImportance
		return
	}

	versionDir := filepath.Join(packageDir, o.selectedVersion)
	logger.Printf("Checking installation status for %s (GPU: %s) at %s\n", o.selectedVersion, o.config.GPUType, versionDir)

	// バージョンディレクトリの存在確認
	if _, err := os.Stat(versionDir); err == nil {
		// GPUタイプに応じた実行ファイルの存在を確認
		isInstalled := o.checkGPUPackageInstalled(versionDir)

		if isInstalled {
			o.installedLabel.SetText("✓ Installed")
			o.installedLabel.Importance = widget.SuccessImportance
			logger.Printf("Version %s (%s) is installed\n", o.selectedVersion, o.config.GPUType)
		} else {
			o.installedLabel.SetText("✗ Wrong GPU Type")
			o.installedLabel.Importance = widget.WarningImportance
			logger.Printf("Version %s exists but wrong GPU type (%s)\n", o.selectedVersion, o.config.GPUType)
		}
	} else {
		o.installedLabel.SetText("✗ Not Installed")
		o.installedLabel.Importance = widget.WarningImportance
		logger.Printf("Version %s not installed\n", o.selectedVersion)
	}
}

// GPUタイプに応じたパッケージがインストールされているか確認
func (o *OrbitApp) checkGPUPackageInstalled(versionDir string) bool {
	// GPU タイプに応じた実行ファイルを探す
	var checkPaths []string

	switch o.config.GPUType {
	case "nvidia":
		checkPaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_nvidia_gpu.bat"),
			filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", ".ci", "windows_nvidia_base_files", "run_nvidia_gpu.bat"),
		}
	case "amd":
		checkPaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_amd_gpu.bat"),
		}
	case "cpu":
		checkPaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_cpu.bat"),
			filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", ".ci", "windows_nvidia_base_files", "run_cpu.bat"),
		}
	}

	// いずれかのパスが存在すればインストール済み
	for _, path := range checkPaths {
		if _, err := os.Stat(path); err == nil {
			logger.Printf("Found GPU-specific file: %s\n", path)
			return true
		}
	}

	return false
}

func (o *OrbitApp) fetchReleases() {
	logger.Println("Fetching ComfyUI releases from GitHub...")
	o.updateStatus("Loading ComfyUI releases from GitHub...")

	resp, err := http.Get(githubAPIURL)
	if err != nil {
		errMsg := fmt.Sprintf("Error fetching releases: %v", err)
		logger.Printf("ERROR: %s\n", errMsg)
		o.updateStatus(errMsg)
		return
	}
	defer resp.Body.Close()

	logger.Printf("HTTP Response Status: %s\n", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Sprintf("Error reading response: %v", err)
		logger.Printf("ERROR: %s\n", errMsg)
		o.updateStatus(errMsg)
		return
	}

	logger.Printf("Response body size: %d bytes\n", len(body))

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		errMsg := fmt.Sprintf("Error parsing releases: %v", err)
		logger.Printf("ERROR: %s\n", errMsg)
		o.updateStatus(errMsg)
		return
	}

	logger.Printf("Successfully parsed %d releases\n", len(releases))
	o.releases = releases

	// バージョンリストを更新
	versions := make([]string, len(releases))
	for i, release := range releases {
		versions[i] = release.TagName
	}

	o.versionSelect.Options = versions
	if len(versions) > 0 {
		o.versionSelect.SetSelected(versions[0])
	}

	o.launchButton.Enable()
	o.updateStatus(fmt.Sprintf("Loaded %d ComfyUI releases", len(releases)))
}

func (o *OrbitApp) updateStatus(message string) {
	o.statusLabel.SetText(message)
}

func (o *OrbitApp) launchComfyUI() {
	version := o.selectedVersion
	logger.Printf("launchComfyUI called for version: %s (GPU: %s)\n", version, o.config.GPUType)
	o.updateStatus(fmt.Sprintf("Launching ComfyUI %s...", version))

	versionDir := filepath.Join(packageDir, version)
	logger.Printf("Checking version directory: %s\n", versionDir)

	// バージョンディレクトリが存在するか確認
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		// ダウンロードと解凍が必要
		logger.Printf("Version directory not found, will download\n")
		o.updateStatus(fmt.Sprintf("Version %s not found, starting download...", version))
		o.downloadAndExtract()
	} else {
		// 既にダウンロード済み、起動
		logger.Printf("Version directory found, proceeding to launch\n")
		o.updateStatus(fmt.Sprintf("Version %s found, starting ComfyUI...", version))
		o.startComfyUI(versionDir, version)
	}
}

func (o *OrbitApp) downloadAndExtract() {
	version := o.selectedVersion

	// リリース情報を探す
	var release *Release
	for i := range o.releases {
		if o.releases[i].TagName == version {
			release = &o.releases[i]
			break
		}
	}

	if release == nil {
		dialog.ShowError(fmt.Errorf("Release %s not found", version), o.window)
		return
	}

	// GPU タイプに応じたダウンロードURLを探す
	var downloadURL string
	var targetFilename string

	switch o.config.GPUType {
	case "nvidia":
		targetFilename = "ComfyUI_windows_portable_nvidia"
	case "amd":
		targetFilename = "ComfyUI_windows_portable_amd"
	case "cpu":
		targetFilename = "ComfyUI_windows_portable_cpu"
	default:
		targetFilename = "ComfyUI_windows_portable_nvidia" // デフォルトはNVIDIA
	}

	for _, asset := range release.Assets {
		// 指定されたGPUタイプのファイルを探す
		if strings.Contains(asset.Name, targetFilename) && strings.HasSuffix(asset.Name, ".7z") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	// 見つからなければwindows_portableで汎用検索（古いバージョン対応）
	if downloadURL == "" {
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, "windows_portable") && strings.HasSuffix(asset.Name, ".7z") {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	if downloadURL == "" {
		dialog.ShowError(fmt.Errorf("ComfyUI Windows portable package not found for %s GPU in release %s",
			strings.ToUpper(o.config.GPUType), version), o.window)
		return
	}

	// プログレスダイアログを表示
	progressBar := widget.NewProgressBarInfinite()
	progressLabel := widget.NewLabel("Preparing download...")
	progressContent := container.NewVBox(progressLabel, progressBar)
	progressDialog := dialog.NewCustom("Downloading", "Cancel", progressContent, o.window)
	progressDialog.Show()

	go func() {
		// tempディレクトリを作成
		os.MkdirAll(tempDir, 0755)

		// tempにダウンロード
		progressLabel.SetText(fmt.Sprintf("Downloading ComfyUI %s (%s) to temp...", version, strings.ToUpper(o.config.GPUType)))
		downloadPath := filepath.Join(tempDir, fmt.Sprintf("%s_%s.7z", version, o.config.GPUType))

		if err := o.downloadFile(downloadURL, downloadPath); err != nil {
			progressDialog.Hide()
			dialog.ShowError(fmt.Errorf("Download failed: %v", err), o.window)
			return
		}

		// packagesディレクトリに解凍
		progressLabel.SetText("Extracting files...")
		versionDir := filepath.Join(packageDir, version)
		os.MkdirAll(packageDir, 0755)

		if err := o.extract7z(downloadPath, versionDir); err != nil {
			progressDialog.Hide()
			dialog.ShowError(fmt.Errorf("Extraction failed: %v", err), o.window)
			return
		}

		// tempのダウンロードファイルを削除
		os.Remove(downloadPath)

		progressDialog.Hide()
		o.updateStatus(fmt.Sprintf("ComfyUI %s installed successfully!", version))
		o.updateInstalledStatus()

		// 起動
		o.startComfyUI(versionDir, version)
	}()
}

func (o *OrbitApp) downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (o *OrbitApp) extract7z(archivePath, destDir string) error {
	// 7zコマンドを使用して解凍
	var cmd *exec.Cmd

	// 7zの場所を探す
	sevenZipPaths := []string{
		"C:\\Program Files\\7-Zip\\7z.exe",
		"C:\\Program Files (x86)\\7-Zip\\7z.exe",
		"7z.exe", // PATH に含まれている場合
	}

	var sevenZipPath string
	for _, path := range sevenZipPaths {
		if _, err := os.Stat(path); err == nil {
			sevenZipPath = path
			break
		}
		if _, err := exec.LookPath(path); err == nil {
			sevenZipPath = path
			break
		}
	}

	if sevenZipPath == "" {
		return fmt.Errorf("7-Zip not found. Please install 7-Zip from https://www.7-zip.org/")
	}

	cmd = exec.Command(sevenZipPath, "x", archivePath, fmt.Sprintf("-o%s", destDir), "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}

	return nil
}

func (o *OrbitApp) startComfyUI(versionDir, version string) {
	logger.Printf("Starting ComfyUI %s (GPU: %s) from %s\n", version, o.config.GPUType, versionDir)

	// ComfyUIの実行ファイルを探す
	var exePath string

	// GPU タイプに応じたパスを構築
	var possiblePaths []string

	switch o.config.GPUType {
	case "nvidia":
		possiblePaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_nvidia_gpu.bat"),
			filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", ".ci", "windows_nvidia_base_files", "run_nvidia_gpu.bat"),
			filepath.Join(versionDir, "run_nvidia_gpu.bat"),
		}
	case "amd":
		possiblePaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_amd_gpu.bat"),
			filepath.Join(versionDir, "run_amd_gpu.bat"),
		}
	case "cpu":
		possiblePaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_cpu.bat"),
			filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", ".ci", "windows_nvidia_base_files", "run_cpu.bat"),
			filepath.Join(versionDir, "run_cpu.bat"),
		}
	default:
		// フォールバック: すべてのパスを試す
		possiblePaths = []string{
			filepath.Join(versionDir, "ComfyUI_windows_portable", "run_nvidia_gpu.bat"),
			filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", ".ci", "windows_nvidia_base_files", "run_nvidia_gpu.bat"),
			filepath.Join(versionDir, "ComfyUI", "main.py"),
			filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", "main.py"),
		}
	}

	logger.Println("Searching for executable in the following paths:")
	for i, path := range possiblePaths {
		logger.Printf("  [%d] %s\n", i+1, path)
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			exePath = path
			logger.Printf("Found executable: %s\n", exePath)
			break
		}
	}

	if exePath == "" {
		errMsg := fmt.Sprintf("ComfyUI executable not found in %s\nTried paths:\n%s", versionDir, strings.Join(possiblePaths, "\n"))
		logger.Printf("ERROR: %s\n", errMsg)
		dialog.ShowError(fmt.Errorf(errMsg), o.window)
		return
	}

	// ComfyUIを起動
	var cmd *exec.Cmd
	var workDir string

	if strings.HasSuffix(exePath, ".bat") {
		// .batファイルの場合、絶対パスで実行
		absPath, _ := filepath.Abs(exePath)
		workDir = filepath.Dir(absPath)

		// .ci内のスクリプトの場合は、ComfyUI_windows_portableディレクトリから実行
		if strings.Contains(exePath, ".ci") {
			workDir = filepath.Join(versionDir, "ComfyUI_windows_portable")
			logger.Printf(".ci script detected, using workdir: %s\n", workDir)
		}

		// batファイルを新しいコマンドプロンプトウィンドウで実行
		batFileName := filepath.Base(absPath)

		// 方法1: start コマンドでbatファイルを直接実行
		cmd = exec.Command("cmd", "/c", "start", "ComfyUI", "/D", workDir, batFileName)
		cmd.Dir = workDir

		cmdStr := fmt.Sprintf("cmd /c start \"ComfyUI\" /D \"%s\" %s", workDir, batFileName)
		logger.Printf("Executing command: %s\n", cmdStr)
		logger.Printf("Working directory: %s\n", workDir)
		logger.Printf("Batch file path: %s\n", absPath)
		o.updateStatus(fmt.Sprintf("Starting ComfyUI from: %s", batFileName))
	} else {
		// Pythonスクリプトの場合
		logger.Printf("Executing Python script: %s\n", exePath)
		cmd = exec.Command("python", exePath)
		workDir = filepath.Dir(exePath)
		cmd.Dir = workDir
	}

	logger.Printf("Starting process...\n")
	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("Failed to launch ComfyUI: %v\nCommand: %v\nWorkDir: %v", err, cmd.Args, workDir)
		logger.Printf("ERROR: %s\n", errMsg)
		dialog.ShowError(fmt.Errorf(errMsg), o.window)
		return
	}

	logger.Printf("Process started successfully (PID: %d)\n", cmd.Process.Pid)

	// 設定を保存
	o.config.LastVersion = version
	o.saveConfig()

	o.updateStatus(fmt.Sprintf("ComfyUI %s launched successfully!", version))
	dialog.ShowInformation("Success", fmt.Sprintf("ComfyUI %s has been launched!", version), o.window)
}

func init() {
	// Windows環境でのみ動作する
	if runtime.GOOS != "windows" {
		fmt.Println("This launcher is designed for Windows only")
		os.Exit(1)
	}
}
