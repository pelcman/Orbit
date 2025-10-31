package main

import (
	"encoding/json"
	"fmt"
	"image/color"
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
	Body    string  `json:"body"`
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
	LastVersion         string      `json:"last_version"`
	CustomApps          []CustomApp `json:"custom_apps"`
	GPUType             string      `json:"gpu_type"`             // "nvidia", "amd", "cpu"
	InstallRequirements bool        `json:"install_requirements"` // プレプロセス: requirements.txtをインストール
	InstallPyTorch      bool        `json:"install_pytorch"`      // プレプロセス: PyTorchをインストール
	RunPreProcess       bool        `json:"run_pre_process"`      // カスタムプレプロセスを実行
	RunPostProcess      bool        `json:"run_post_process"`     // カスタムポストプロセスを実行
	PreProcessCommand   string      `json:"pre_process_command"`  // カスタムプレプロセスコマンド
	PostProcessCommand  string      `json:"post_process_command"` // カスタムポストプロセスコマンド
	GoogleFontURL       string      `json:"google_font_url"`      // Google FontsのURL
	FontWeight          int         `json:"font_weight"`          // フォントの太さ (100-900, デフォルト700=Bold)
}

type OrbitApp struct {
	app                    fyne.App
	window                 fyne.Window
	releases               []Release
	config                 Config
	statusLabel            *widget.Label
	versionSelect          *widget.Select
	installedLabel         *widget.Label
	launchButton           *widget.Button
	gpuSelect              *widget.Select
	selectedVersion        string
	installRequirementsChk *widget.Check
	customAppButtons       []*CustomAppButton
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
	orbitApp.window.Resize(fyne.NewSize(520, 730))
	orbitApp.window.SetFixedSize(false)

	// アイコンを設定
	iconPath := "Img/icon.png"
	if iconResource, err := fyne.LoadResourceFromPath(iconPath); err == nil {
		orbitApp.window.SetIcon(iconResource)
		logger.Printf("Window icon set: %s\n", iconPath)
	} else {
		logger.Printf("Failed to load icon: %v\n", err)
	}

	logger.Println("Loading configuration...")
	orbitApp.loadConfig()

	logger.Println("Loading custom font...")
	orbitApp.loadCustomFont()

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

	// Google Fontのデフォルト設定（Nunito）
	if o.config.GoogleFontURL == "" {
		o.config.GoogleFontURL = "https://fonts.google.com/download?family=Nunito"
		o.saveConfig()
	}

	// フォントウェイトのデフォルト設定（Bold = 700）
	if o.config.FontWeight == 0 {
		o.config.FontWeight = 700
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

// フォントウェイトに対応する名前を取得
func (o *OrbitApp) getFontWeightName(weight int) string {
	switch {
	case weight <= 100:
		return "Thin"
	case weight <= 200:
		return "ExtraLight"
	case weight <= 300:
		return "Light"
	case weight <= 400:
		return "Regular"
	case weight <= 500:
		return "Medium"
	case weight <= 600:
		return "SemiBold"
	case weight <= 700:
		return "Bold"
	case weight <= 800:
		return "ExtraBold"
	default:
		return "Black"
	}
}

// Google Fontをダウンロードしてカスタムフォントとして適用
func (o *OrbitApp) loadCustomFont() {
	if o.config.GoogleFontURL == "" {
		logger.Println("No Google Font URL configured, using default font")
		return
	}

	// フォントキャッシュディレクトリを作成
	fontCacheDir := filepath.Join(tempDir, "font_cache")
	os.MkdirAll(fontCacheDir, 0755)

	// URLからフォント名を抽出（簡易的な方法）
	fontName := "CustomFont"
	if strings.Contains(o.config.GoogleFontURL, "family=") {
		parts := strings.Split(o.config.GoogleFontURL, "family=")
		if len(parts) > 1 {
			fontName = strings.Split(parts[1], "&")[0]
		}
	}

	// キャッシュファイルパス（.ttfファイル）- ウェイト別にキャッシュ
	fontPath := filepath.Join(fontCacheDir, fmt.Sprintf("%s_%d.ttf", fontName, o.config.FontWeight))

	// キャッシュが存在しない場合はダウンロード
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		logger.Printf("Downloading font: %s\n", fontName)

		// Google Fonts GitHubリポジトリから直接.ttfファイルを取得
		// フォント名を小文字に変換してURLを構築
		fontNameLower := strings.ToLower(fontName)

		// フォントウェイトに応じたファイル名を決定
		weightName := o.getFontWeightName(o.config.FontWeight)

		// 複数のURL候補を試す
		possibleURLs := []string{
			// Variable Font (最新のGoogle Fontsはこの形式) - すべてのウェイトをサポート
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ofl/%s/%s%%5Bwght%%5D.ttf", fontNameLower, fontName),
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ofl/%s/%s[wght].ttf", fontNameLower, fontName),
			// Static Font - 指定されたウェイト
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ofl/%s/%s-%s.ttf", fontNameLower, fontName, weightName),
			// Static Font - Bold (700)
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ofl/%s/%s-Bold.ttf", fontNameLower, fontName),
			// Static Font - Regular (400)
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ofl/%s/%s-Regular.ttf", fontNameLower, fontName),
			// Static Font - 小文字
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ofl/%s/%s-Regular.ttf", fontNameLower, fontNameLower),
			// Apache License
			fmt.Sprintf("https://github.com/google/fonts/raw/main/apache/%s/%s%%5Bwght%%5D.ttf", fontNameLower, fontName),
			fmt.Sprintf("https://github.com/google/fonts/raw/main/apache/%s/%s-%s.ttf", fontNameLower, fontName, weightName),
			fmt.Sprintf("https://github.com/google/fonts/raw/main/apache/%s/%s-Bold.ttf", fontNameLower, fontName),
			fmt.Sprintf("https://github.com/google/fonts/raw/main/apache/%s/%s-Regular.ttf", fontNameLower, fontName),
			// UFL
			fmt.Sprintf("https://github.com/google/fonts/raw/main/ufl/%s/%s-Regular.ttf", fontNameLower, fontName),
		}

		var resp *http.Response
		var downloadErr error
		var successURL string

		for _, url := range possibleURLs {
			logger.Printf("Trying URL: %s\n", url)

			// HTTPクライアントを作成してリダイレクトを許可
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return nil
				},
			}

			resp, downloadErr = client.Get(url)
			if downloadErr == nil && resp.StatusCode == http.StatusOK {
				successURL = url
				logger.Printf("Successfully found font at: %s (Status: %d)\n", url, resp.StatusCode)
				break
			}
			if resp != nil {
				logger.Printf("Failed: Status %d\n", resp.StatusCode)
				resp.Body.Close()
			}
		}

		if successURL == "" {
			logger.Printf("Failed to download font from any URL, using default font\n")
			return
		}
		defer resp.Body.Close()

		// フォントファイルを保存
		out, err := os.Create(fontPath)
		if err != nil {
			logger.Printf("Failed to create font file: %v\n", err)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			logger.Printf("Failed to save font file: %v\n", err)
			return
		}

		logger.Printf("Font downloaded successfully: %s\n", fontPath)
	} else {
		logger.Printf("Using cached font: %s\n", fontPath)
	}

	// Fyneアプリケーションにカスタムフォントを設定
	// 注: Fyneでカスタムフォントを設定するには、カスタムテーマを作成する必要があります
	customTheme := &customFontTheme{
		fontPath:   fontPath,
		fontWeight: o.config.FontWeight,
	}
	o.app.Settings().SetTheme(customTheme)
	logger.Printf("Custom font applied successfully (weight: %d)\n", o.config.FontWeight)
}

// カスタムフォントテーマ
type customFontTheme struct {
	fontPath   string
	fontWeight int
}

func (t *customFontTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// グラスモーフィズム風の半透明カラー
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 25, G: 25, B: 35, A: 250}
	case theme.ColorNameButton:
		return color.NRGBA{R: 60, G: 60, B: 80, A: 200}
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 40, G: 40, B: 50, A: 150}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 100, G: 100, B: 110, A: 180}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 45, G: 45, B: 60, A: 220}
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 20, G: 20, B: 30, A: 230}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *customFontTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *customFontTheme) Font(style fyne.TextStyle) fyne.Resource {
	// カスタムフォントファイルを読み込む
	if data, err := os.ReadFile(t.fontPath); err == nil {
		// 指定されたウェイトのフォントを全てのテキストに適用
		// Boldフラグに関係なく、設定されたウェイトのフォントを返す
		return fyne.NewStaticResource("CustomFont.ttf", data)
	}
	// フォールバック: デフォルトフォント
	// ウェイトに応じてBoldスタイルを適用
	if t.fontWeight >= 700 {
		style.Bold = true
	}
	return theme.DefaultTheme().Font(style)
}

func (t *customFontTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func (o *OrbitApp) setupModernUI() {
	// === 上部: ロゴ ===
	banner := canvas.NewImageFromFile("Img/banner.png")
	banner.FillMode = canvas.ImageFillContain
	banner.SetMinSize(fyne.NewSize(275, 115))

	// === 左下: インストール済みバージョン選択 ===
	o.versionSelect = widget.NewSelect([]string{}, func(value string) {
		o.selectedVersion = value
		o.updateInstalledStatus()
	})
	o.versionSelect.PlaceHolder = "Select Installed Version"

	o.installedLabel = widget.NewLabel("No versions installed")
	o.installedLabel.TextStyle = fyne.TextStyle{Bold: o.config.FontWeight >= 700}

	// インストールボタン
	installButton := widget.NewButton("Install New Version", func() {
		o.showInstallDialog()
	})
	installButton.Importance = widget.LowImportance

	versionCard := container.NewVBox(
		widget.NewLabel("Installed Versions:"),
		o.versionSelect,
		installButton,
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
	detectedLabel.TextStyle = fyne.TextStyle{Italic: true, Bold: o.config.FontWeight >= 700}

	// プレプロセスオプション: requirements.txtインストール
	o.installRequirementsChk = widget.NewCheck("Install ComfyUI requirements", func(checked bool) {
		o.config.InstallRequirements = checked
		o.saveConfig()
	})
	o.installRequirementsChk.SetChecked(o.config.InstallRequirements)

	// プレプロセスオプション: PyTorchインストール
	installPyTorchChk := widget.NewCheck("Install PyTorch with CUDA", func(checked bool) {
		o.config.InstallPyTorch = checked
		o.saveConfig()
	})
	installPyTorchChk.SetChecked(o.config.InstallPyTorch)

	optionsCard := container.NewVBox(
		widget.NewLabel("GPU Type:"),
		o.gpuSelect,
		detectedLabel,
		widget.NewSeparator(),
		widget.NewLabel("Launch Options:"),
		o.installRequirementsChk,
		installPyTorchChk,
	)

	// 左右のカードを横並び（中央揃え）
	middleSection := container.NewCenter(
		container.NewPadded(
			container.NewGridWithColumns(2,
				versionCard,
				optionsCard,
			),
		),
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

	buttonRow := container.NewHBox(
		layout.NewSpacer(),
		o.launchButton,
		layout.NewSpacer(),
	)

	// === ステータスバー ===
	o.statusLabel = widget.NewLabel("Select an installed version or install a new one")
	o.statusLabel.Wrapping = fyne.TextWrapWord
	o.statusLabel.TextStyle = fyne.TextStyle{Italic: true, Bold: o.config.FontWeight >= 700}

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

	// インストール済みバージョンを読み込む
	go o.loadInstalledVersions()
}

// インストール済みバージョンを読み込む
func (o *OrbitApp) loadInstalledVersions() {
	logger.Println("Loading installed versions from packages directory...")
	o.updateStatus("Loading installed versions...")

	// packagesディレクトリを作成（存在しない場合）
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		logger.Printf("Failed to create packages directory: %v\n", err)
		o.updateStatus("Failed to load versions")
		return
	}

	// packagesディレクトリ内のサブディレクトリを列挙
	entries, err := os.ReadDir(packageDir)
	if err != nil {
		logger.Printf("Failed to read packages directory: %v\n", err)
		o.updateStatus("Failed to load versions")
		return
	}

	var installedVersions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// バージョン名を追加
			installedVersions = append(installedVersions, entry.Name())
		}
	}

	logger.Printf("Found %d installed versions: %v\n", len(installedVersions), installedVersions)

	if len(installedVersions) == 0 {
		o.versionSelect.Options = []string{}
		o.versionSelect.PlaceHolder = "No versions installed"
		o.updateStatus("No versions installed. Click 'Install New Version' to get started.")
		o.launchButton.Disable()
		return
	}

	// バージョンリストを更新
	o.versionSelect.Options = installedVersions
	o.versionSelect.PlaceHolder = "Select Installed Version"
	if len(installedVersions) > 0 {
		o.versionSelect.SetSelected(installedVersions[0])
	}

	o.launchButton.Enable()
	o.updateStatus(fmt.Sprintf("Found %d installed version(s)", len(installedVersions)))
}

// インストールダイアログを表示
func (o *OrbitApp) showInstallDialog() {
	logger.Println("Opening install dialog...")

	// プログレスバーとステータスラベル
	progressBar := widget.NewProgressBarInfinite()
	progressLabel := widget.NewLabel("Loading available versions from GitHub...")

	// 選択されたバージョン
	var selectedVersion string

	// バージョンリストコンテナ（後で更新）
	versionListContainer := container.NewVBox()

	// ページネーション変数
	currentPage := 1
	itemsPerPage := 30

	// ページ情報ラベル
	pageLabel := widget.NewLabel("Page 1")

	// 前のページボタン
	prevButton := widget.NewButton("Previous", nil)
	prevButton.Disable()

	// 次のページボタン
	nextButton := widget.NewButton("  Next  ", nil)
	nextButton.Disable()

	// GPU選択
	gpuSelect := widget.NewSelect([]string{"NVIDIA GPU", "AMD GPU", "CPU"}, nil)
	gpuSelect.PlaceHolder = "Select GPU Type"
	switch o.config.GPUType {
	case "nvidia":
		gpuSelect.SetSelected("NVIDIA GPU")
	case "amd":
		gpuSelect.SetSelected("AMD GPU")
	case "cpu":
		gpuSelect.SetSelected("CPU")
	}
	gpuSelect.Disable()

	// What's Changedラベル
	changesLabel := widget.NewLabel("Select a version to see changes")
	changesLabel.Wrapping = fyne.TextWrapWord
	changesLabel.TextStyle = fyne.TextStyle{Italic: true}

	// What's Changedスクロールエリア
	changesScroll := container.NewVScroll(changesLabel)
	changesScroll.SetMinSize(fyne.NewSize(200, 200))

	// インストールボタン
	installBtn := widget.NewButton("Install", func() {
		if selectedVersion == "" {
			dialog.ShowError(fmt.Errorf("Please select a version"), o.window)
			return
		}

		selectedGPU := gpuSelect.Selected
		if selectedGPU == "" {
			dialog.ShowError(fmt.Errorf("Please select GPU type"), o.window)
			return
		}

		// GPU タイプを決定
		var gpuType string
		switch selectedGPU {
		case "NVIDIA GPU":
			gpuType = "nvidia"
		case "AMD GPU":
			gpuType = "amd"
		case "CPU":
			gpuType = "cpu"
		default:
			gpuType = o.config.GPUType
		}

		// インストール処理を開始
		o.startInstallation(selectedVersion, gpuType)
	})
	installBtn.Importance = widget.HighImportance
	installBtn.Disable()

	// Closeボタン
	closeBtn := widget.NewButton("Close", nil)

	// バージョンリストを更新する関数
	updateVersionList := func() {
		versionListContainer.Objects = nil

		// インストール済みバージョンのマップを作成
		installedVersions := make(map[string]bool)
		entries, err := os.ReadDir(packageDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					installedVersions[entry.Name()] = true
				}
			}
		}

		// ページ範囲を計算
		startIdx := (currentPage - 1) * itemsPerPage
		endIdx := startIdx + itemsPerPage
		if endIdx > len(o.releases) {
			endIdx = len(o.releases)
		}

		// バージョンリストを作成
		for i := startIdx; i < endIdx; i++ {
			release := o.releases[i]
			version := release.TagName
			isInstalled := installedVersions[version]

			// クロージャ用に変数をキャプチャ
			v := version
			rel := release

			// ラジオボタン風のボタンを作成（横幅を小さく）
			versionBtn := widget.NewButton(v, nil)

			if isInstalled {
				// インストール済みの場合はグレーアウト
				versionBtn.Importance = widget.LowImportance
				versionBtn.Disable()
			} else {
				// 未インストールの場合は選択可能
				versionBtn.Importance = widget.MediumImportance
				versionBtn.OnTapped = func() {
					selectedVersion = v
					logger.Printf("Selected version: %s\n", v)

					// What's Changedを更新
					if rel.Body != "" {
						// リリースノートを整形（最初の500文字まで）
						changes := rel.Body
						if len(changes) > 500 {
							changes = changes[:500] + "..."
						}
						changesLabel.SetText(changes)
					} else {
						changesLabel.SetText("No release notes available")
					}
					changesLabel.Refresh()

					// 選択状態を視覚的に表示
					for _, obj := range versionListContainer.Objects {
						if btn, ok := obj.(*widget.Button); ok {
							if btn.Text == v {
								btn.Importance = widget.HighImportance
							} else if !btn.Disabled() {
								btn.Importance = widget.MediumImportance
							}
							btn.Refresh()
						}
					}
				}
			}

			versionListContainer.Add(versionBtn)
		}

		// ページ情報を更新
		totalPages := (len(o.releases) + itemsPerPage - 1) / itemsPerPage
		pageLabel.SetText(fmt.Sprintf("Page %d / %d", currentPage, totalPages))

		// ボタンの有効/無効を更新
		if currentPage > 1 {
			prevButton.Enable()
		} else {
			prevButton.Disable()
		}

		if currentPage < totalPages {
			nextButton.Enable()
		} else {
			nextButton.Disable()
		}

		versionListContainer.Refresh()
	}

	// ページボタンのハンドラを設定
	prevButton.OnTapped = func() {
		if currentPage > 1 {
			currentPage--
			updateVersionList()
		}
	}

	nextButton.OnTapped = func() {
		totalPages := (len(o.releases) + itemsPerPage - 1) / itemsPerPage
		if currentPage < totalPages {
			currentPage++
			updateVersionList()
		}
	}

	// 左側: バージョンリスト（横幅を小さく）
	versionScroll := container.NewVScroll(versionListContainer)
	versionScroll.SetMinSize(fyne.NewSize(180, 450))

	paginationButtons := container.NewHBox(
		prevButton,
		layout.NewSpacer(),
		pageLabel,
		layout.NewSpacer(),
		nextButton,
	)

	leftPanel := container.NewBorder(
		container.NewVBox(
			progressLabel,
			progressBar,
		),
		container.NewPadded(paginationButtons),
		nil, nil,
		versionScroll,
	)

	// 右側: GPU選択、What's Changed、ボタン（縦に並べる）
	rightPanel := container.NewBorder(
		nil,
		container.NewVBox(
			widget.NewSeparator(),
			container.NewPadded(installBtn),
			container.NewPadded(closeBtn),
		),
		nil, nil,
		container.NewVBox(
			widget.NewLabel("Select GPU Type:"),
			gpuSelect,
			widget.NewSeparator(),
			widget.NewLabel("What's Changed:"),
			changesScroll,
		),
	)

	// 背景を作成（角丸・半透明）
	background := canvas.NewRectangle(color.NRGBA{R: 40, G: 40, B: 50, A: 240})
	background.CornerRadius = 12

	// メインレイアウト
	installContent := container.NewStack(
		background,
		container.NewPadded(
			container.NewBorder(
				nil, nil,
				leftPanel,
				container.NewPadded(rightPanel),
				nil,
			),
		),
	)

	// ダイアログを作成（Dismissボタンなし）
	installDialog := dialog.NewCustomWithoutButtons("Install ComfyUI", installContent, o.window)
	installDialog.Resize(fyne.NewSize(700, 600))

	// Closeボタンにハンドラを設定
	closeBtn.OnTapped = func() {
		installDialog.Hide()
	}

	installDialog.Show()

	// バックグラウンドでリリースを取得
	go func() {
		o.fetchReleases()

		// リリース取得完了後、UIを更新
		if len(o.releases) == 0 {
			progressLabel.SetText("Failed to load releases from GitHub")
			progressBar.Hide()
			return
		}

		progressLabel.SetText(fmt.Sprintf("Loaded %d versions", len(o.releases)))
		progressBar.Hide()

		// バージョンリストを更新
		updateVersionList()

		gpuSelect.Enable()
		installBtn.Enable()
	}()
}

// インストール処理を開始
func (o *OrbitApp) startInstallation(version, gpuType string) {
	logger.Printf("Starting installation: version=%s, gpuType=%s\n", version, gpuType)
	o.updateStatus(fmt.Sprintf("Installing ComfyUI %s (%s)...", version, strings.ToUpper(gpuType)))

	// 既にインストール済みか確認
	versionDir := filepath.Join(packageDir, version)
	if _, err := os.Stat(versionDir); err == nil {
		dialog.ShowInformation("Already Installed",
			fmt.Sprintf("ComfyUI %s is already installed.\nVersion directory: %s", version, versionDir),
			o.window)
		return
	}

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

	switch gpuType {
	case "nvidia":
		targetFilename = "ComfyUI_windows_portable_nvidia"
	case "amd":
		targetFilename = "ComfyUI_windows_portable_amd"
	case "cpu":
		targetFilename = "ComfyUI_windows_portable_cpu"
	default:
		targetFilename = "ComfyUI_windows_portable_nvidia"
	}

	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, targetFilename) && strings.HasSuffix(asset.Name, ".7z") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	// 見つからなければwindows_portableで汎用検索
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
			strings.ToUpper(gpuType), version), o.window)
		return
	}

	// プログレスダイアログを表示
	progressBar := widget.NewProgressBarInfinite()
	progressLabel := widget.NewLabel("Preparing download...")
	progressContent := container.NewVBox(progressLabel, progressBar)
	progressDialog := dialog.NewCustom("Installing", "Cancel", progressContent, o.window)
	progressDialog.Show()

	go func() {
		// tempディレクトリを作成
		os.MkdirAll(tempDir, 0755)

		// tempにダウンロード
		progressLabel.SetText(fmt.Sprintf("Downloading ComfyUI %s (%s)...", version, strings.ToUpper(gpuType)))
		downloadPath := filepath.Join(tempDir, fmt.Sprintf("%s_%s.7z", version, gpuType))

		if err := o.downloadFile(downloadURL, downloadPath); err != nil {
			progressDialog.Hide()
			dialog.ShowError(fmt.Errorf("Download failed: %v", err), o.window)
			return
		}

		// packagesディレクトリに解凍
		progressLabel.SetText("Extracting files...")
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

		// インストール済みバージョンを再読み込み
		o.loadInstalledVersions()

		dialog.ShowInformation("Installation Complete",
			fmt.Sprintf("ComfyUI %s has been installed successfully!\n\nYou can now select it from the installed versions list.", version),
			o.window)
	}()
}

// CustomAppButton - カスタムアプリボタンウィジェット
type CustomAppButton struct {
	widget.BaseWidget
	app          *OrbitApp
	index        int
	icon         *canvas.Image
	label        *canvas.Text
	background   *canvas.Rectangle
	onTapped     func()
	onRightClick func()
}

func NewCustomAppButton(app *OrbitApp, index int, onTapped, onRightClick func()) *CustomAppButton {
	btn := &CustomAppButton{
		app:          app,
		index:        index,
		onTapped:     onTapped,
		onRightClick: onRightClick,
	}
	btn.ExtendBaseWidget(btn)
	return btn
}

func (b *CustomAppButton) CreateRenderer() fyne.WidgetRenderer {
	// 背景
	b.background = canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 50, A: 255})

	// アイコン画像
	b.icon = canvas.NewImageFromResource(theme.DocumentIcon())
	b.icon.FillMode = canvas.ImageFillContain

	// ラベル
	b.label = canvas.NewText(b.app.config.CustomApps[b.index].Name, color.White)
	b.label.Alignment = fyne.TextAlignCenter
	b.label.TextSize = 9

	// アイコンを更新
	b.updateIcon()

	return &customAppButtonRenderer{
		button:     b,
		background: b.background,
		icon:       b.icon,
		label:      b.label,
	}
}

func (b *CustomAppButton) Tapped(_ *fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *CustomAppButton) TappedSecondary(_ *fyne.PointEvent) {
	if b.onRightClick != nil {
		b.onRightClick()
	}
}

func (b *CustomAppButton) updateIcon() {
	appPath := b.app.config.CustomApps[b.index].Path

	if appPath != "" && b.icon != nil {
		// ファイルが存在する場合、拡張子に応じたアイコンを表示
		if _, err := os.Stat(appPath); err == nil {
			ext := strings.ToLower(filepath.Ext(appPath))

			// Windowsの実行ファイルからアイコンを抽出して表示
			if iconPath := extractIconFromExe(appPath); iconPath != "" {
				if img := canvas.NewImageFromFile(iconPath); img != nil {
					b.icon = img
					b.icon.FillMode = canvas.ImageFillContain
					logger.Printf("Loaded icon from: %s\n", iconPath)
				}
			} else {
				// アイコン抽出に失敗した場合は拡張子に応じたアイコン
				switch ext {
				case ".exe", ".bat", ".cmd":
					b.icon.Resource = theme.ComputerIcon()
				case ".lnk":
					b.icon.Resource = theme.FileIcon()
				default:
					b.icon.Resource = theme.FileApplicationIcon()
				}
			}
		} else {
			// ファイルが存在しない場合はデフォルトアイコン
			b.icon.Resource = theme.DocumentIcon()
		}
	} else {
		// パスが設定されていない場合
		b.icon.Resource = theme.DocumentIcon()
	}
	b.icon.Refresh()
}

// Windowsの実行ファイルからアイコンを抽出する
func extractIconFromExe(exePath string) string {
	if runtime.GOOS != "windows" {
		return ""
	}

	// アイコンキャッシュディレクトリを作成
	cacheDir := filepath.Join("temp", "icon_cache")
	os.MkdirAll(cacheDir, 0755)

	// exeファイルのハッシュ値からキャッシュファイル名を生成
	exeBasename := filepath.Base(exePath)
	iconCachePath := filepath.Join(cacheDir, strings.TrimSuffix(exeBasename, filepath.Ext(exeBasename))+".png")

	// キャッシュが存在する場合はそれを使用
	if _, err := os.Stat(iconCachePath); err == nil {
		return iconCachePath
	}

	// PowerShellを使用してアイコンを抽出
	psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Drawing
$icon = [System.Drawing.Icon]::ExtractAssociatedIcon('%s')
if ($icon -ne $null) {
    $bitmap = $icon.ToBitmap()
    $bitmap.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
    $bitmap.Dispose()
    $icon.Dispose()
    Write-Host 'Success'
} else {
    Write-Host 'Failed'
}
`, strings.ReplaceAll(exePath, "'", "''"), strings.ReplaceAll(iconCachePath, "'", "''"))

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Printf("Failed to extract icon from %s: %v\n", exePath, err)
		return ""
	}

	if strings.Contains(string(output), "Success") {
		logger.Printf("Successfully extracted icon to: %s\n", iconCachePath)
		return iconCachePath
	}

	return ""
}

func (b *CustomAppButton) Refresh() {
	b.label.Text = b.app.config.CustomApps[b.index].Name
	b.updateIcon()
	b.BaseWidget.Refresh()
}

// customAppButtonRenderer - カスタムレンダラー
type customAppButtonRenderer struct {
	button     *CustomAppButton
	background *canvas.Rectangle
	icon       *canvas.Image
	label      *canvas.Text
}

func (r *customAppButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)

	// アイコンは32x32固定サイズで中央上部に配置
	iconSize := fyne.NewSize(32, 32)
	iconPos := fyne.NewPos((size.Width-iconSize.Width)/2, 8)
	r.icon.Resize(iconSize)
	r.icon.Move(iconPos)

	// ラベルは下部に配置
	labelHeight := float32(16)
	labelPos := fyne.NewPos(0, size.Height-labelHeight-4)
	r.label.Resize(fyne.NewSize(size.Width, labelHeight))
	r.label.Move(labelPos)
}

func (r *customAppButtonRenderer) MinSize() fyne.Size {
	return fyne.NewSize(70, 60)
}

func (r *customAppButtonRenderer) Refresh() {
	r.background.Refresh()
	r.icon.Refresh()
	r.label.Refresh()
}

func (r *customAppButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.icon, r.label}
}

func (r *customAppButtonRenderer) Destroy() {}

func (o *OrbitApp) createCustomAppIcons() *fyne.Container {
	// 6個のアプリボタンを6列のグリッドで配置
	buttons := make([]fyne.CanvasObject, 6)
	o.customAppButtons = make([]*CustomAppButton, 6)

	for i := 0; i < 6; i++ {
		idx := i
		btn := NewCustomAppButton(o, idx,
			func() {
				// 左クリック: アプリを起動
				o.launchCustomApp(idx)
			},
			func() {
				// 右クリック: 設定を表示
				o.showCustomAppSettings(idx)
			},
		)
		buttons[i] = btn
		o.customAppButtons[i] = btn
	}

	grid := container.NewGridWithColumns(6, buttons...)

	// 中央揃えでラベルとグリッドを配置
	return container.NewVBox(
		container.NewCenter(widget.NewLabel("Custom Apps:")),
		container.NewCenter(
			container.NewPadded(grid),
		),
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

			// ボタンを更新
			if index < len(o.customAppButtons) && o.customAppButtons[index] != nil {
				o.customAppButtons[index].Refresh()
			}
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

// GitHubから全リリースを取得（ページネーション対応）
func (o *OrbitApp) fetchReleases() {
	logger.Println("Fetching ComfyUI releases from GitHub...")

	var allReleases []Release
	page := 1
	perPage := 100 // 1ページあたり100件取得

	for {
		// ページネーション付きURL
		url := fmt.Sprintf("%s?page=%d&per_page=%d", githubAPIURL, page, perPage)
		logger.Printf("Fetching page %d: %s\n", page, url)

		resp, err := http.Get(url)
		if err != nil {
			errMsg := fmt.Sprintf("Error fetching releases: %v", err)
			logger.Printf("ERROR: %s\n", errMsg)
			return
		}

		logger.Printf("HTTP Response Status: %s\n", resp.Status)

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			errMsg := fmt.Sprintf("Error reading response: %v", err)
			logger.Printf("ERROR: %s\n", errMsg)
			return
		}

		var releases []Release
		if err := json.Unmarshal(body, &releases); err != nil {
			errMsg := fmt.Sprintf("Error parsing releases: %v", err)
			logger.Printf("ERROR: %s\n", errMsg)
			return
		}

		logger.Printf("Page %d: parsed %d releases\n", page, len(releases))

		if len(releases) == 0 {
			break // これ以上リリースがない
		}

		allReleases = append(allReleases, releases...)

		// 取得したリリース数がperPageより少ない場合、最後のページに到達
		if len(releases) < perPage {
			break
		}

		page++
	}

	logger.Printf("Successfully fetched %d total releases\n", len(allReleases))
	o.releases = allReleases
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
		// インストールされていない
		logger.Printf("Version directory not found\n")
		dialog.ShowError(fmt.Errorf("Version %s is not installed.\nPlease install it first using 'Install New Version' button.", version), o.window)
		return
	}

	// 既にインストール済み、起動
	logger.Printf("Version directory found, proceeding to launch\n")
	o.updateStatus(fmt.Sprintf("Version %s found, starting ComfyUI...", version))
	o.startComfyUI(versionDir, version)
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

// プレプロセスを実行
func (o *OrbitApp) runPreProcess(versionDir string) error {
	logger.Println("Running pre-process tasks...")

	// PyTorchをインストール（requirements.txtより先に実行）
	if o.config.InstallPyTorch {
		logger.Println("Installing PyTorch with CUDA support...")
		o.updateStatus("Installing PyTorch with CUDA...")

		// Pythonのパスを探す
		pythonPath := filepath.Join(versionDir, "ComfyUI_windows_portable", "python_embeded", "python.exe")
		if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
			// システムのPythonを使用
			pythonPath = "python"
			logger.Println("Using system Python for PyTorch installation")
		} else {
			logger.Printf("Using embedded Python for PyTorch installation: %s\n", pythonPath)
		}

		// 絶対パスを取得
		absPythonPath, _ := filepath.Abs(pythonPath)

		// バッチファイルを一時的に作成して実行
		tempBatPath := filepath.Join(tempDir, "install_pytorch.bat")
		os.MkdirAll(tempDir, 0755)

		// PyTorchのインストールコマンド（CUDA 12.1対応）
		// 公式推奨: https://pytorch.org/get-started/locally/
		var pipCommand string
		switch o.config.GPUType {
		case "nvidia":
			pipCommand = `"%s" -m pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121`
		case "amd":
			// AMD ROCmサポート
			pipCommand = `"%s" -m pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/rocm5.7`
		case "cpu":
			// CPU版
			pipCommand = `"%s" -m pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu`
		default:
			pipCommand = `"%s" -m pip install torch torchvision torchaudio`
		}

		// バッチファイルの内容
		batContent := fmt.Sprintf(`@echo off
echo ========================================
echo Installing PyTorch with CUDA Support
echo ========================================
echo.
echo Python: %s
echo GPU Type: %s
echo.
`+pipCommand+`
echo.
if errorlevel 1 (
    echo ========================================
    echo PyTorch Installation FAILED!
    echo ========================================
    echo Please check the error messages above.
    echo You can close this window when done.
    echo ========================================
) else (
    echo ========================================
    echo PyTorch Installation COMPLETED!
    echo ========================================
    echo You can close this window now.
    echo ========================================
)
echo.
pause
`, absPythonPath, strings.ToUpper(o.config.GPUType), absPythonPath)

		// バッチファイルを書き込み
		if err := os.WriteFile(tempBatPath, []byte(batContent), 0644); err != nil {
			logger.Printf("Failed to create PyTorch batch file: %v\n", err)
			return fmt.Errorf("failed to create PyTorch batch file: %v", err)
		}

		logger.Printf("Created PyTorch installation batch file: %s\n", tempBatPath)

		// バッチファイルを別ウィンドウで実行（同期、完了を待つ）
		startCmd := exec.Command("cmd", "/c", "start", "/wait", "Installing PyTorch", tempBatPath)
		if err := startCmd.Run(); err != nil {
			logger.Printf("Failed to install PyTorch: %v\n", err)
			os.Remove(tempBatPath)
			return fmt.Errorf("failed to install PyTorch: %v", err)
		}

		logger.Println("PyTorch installation completed")
		o.updateStatus("PyTorch installation completed")

		// バッチファイルを削除
		os.Remove(tempBatPath)
	}

	// requirements.txtをインストール
	if o.config.InstallRequirements {
		logger.Println("Installing requirements.txt...")
		o.updateStatus("Installing requirements.txt...")

		// ComfyUIディレクトリ内のrequirements.txtを探す
		requirementsPath := filepath.Join(versionDir, "ComfyUI_windows_portable", "ComfyUI", "requirements.txt")
		if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
			// 代替パスを試す
			requirementsPath = filepath.Join(versionDir, "ComfyUI", "requirements.txt")
			if _, err := os.Stat(requirementsPath); os.IsNotExist(err) {
				logger.Printf("requirements.txt not found in expected locations\n")
				return fmt.Errorf("requirements.txt not found")
			}
		}

		logger.Printf("Found requirements.txt at: %s\n", requirementsPath)

		// Pythonのパスを探す
		pythonPath := filepath.Join(versionDir, "ComfyUI_windows_portable", "python_embeded", "python.exe")
		if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
			// システムのPythonを使用
			pythonPath = "python"
			logger.Println("Using system Python")
		} else {
			logger.Printf("Using embedded Python: %s\n", pythonPath)
		}

		// pip install -r requirements.txt を実行
		// 絶対パスを取得
		absPythonPath, _ := filepath.Abs(pythonPath)
		absRequirementsPath, _ := filepath.Abs(requirementsPath)
		workDir := filepath.Dir(absRequirementsPath)

		// バッチファイルを一時的に作成して実行
		tempBatPath := filepath.Join(tempDir, "install_requirements.bat")
		os.MkdirAll(tempDir, 0755)

		// バッチファイルの内容（別ウィンドウで表示、手動で閉じる）
		batContent := fmt.Sprintf(`@echo off
echo ========================================
echo Installing ComfyUI Requirements
echo ========================================
echo.
echo Python: %s
echo Requirements: %s
echo Working Directory: %s
echo.
cd /d "%s"
"%s" -m pip install -r "%s"
echo.
if errorlevel 1 (
    echo ========================================
    echo Installation FAILED!
    echo ========================================
    echo Please check the error messages above.
    echo You can close this window when done.
    echo ========================================
) else (
    echo ========================================
    echo Installation COMPLETED successfully!
    echo ========================================
    echo You can close this window now.
    echo ========================================
)
echo.
pause
`, absPythonPath, absRequirementsPath, workDir, workDir, absPythonPath, absRequirementsPath)

		// バッチファイルを書き込み
		if err := os.WriteFile(tempBatPath, []byte(batContent), 0644); err != nil {
			logger.Printf("Failed to create batch file: %v\n", err)
			return fmt.Errorf("failed to create batch file: %v", err)
		}

		logger.Printf("Created temporary batch file: %s\n", tempBatPath)
		logger.Printf("Python path: %s\n", absPythonPath)
		logger.Printf("Requirements path: %s\n", absRequirementsPath)
		logger.Printf("Working directory: %s\n", workDir)

		// バッチファイルを別ウィンドウで実行（同期、完了を待つ）
		startCmd := exec.Command("cmd", "/c", "start", "/wait", "Installing Requirements", tempBatPath)
		if err := startCmd.Run(); err != nil {
			logger.Printf("Failed to start installation window: %v\n", err)
			os.Remove(tempBatPath)
			return fmt.Errorf("failed to start installation window: %v", err)
		}

		logger.Println("Requirements installation completed")
		o.updateStatus("Requirements installation completed")

		// バッチファイルを削除
		os.Remove(tempBatPath)
	}

	// カスタムプレプロセス
	if o.config.RunPreProcess && o.config.PreProcessCommand != "" {
		logger.Printf("Running custom pre-process: %s\n", o.config.PreProcessCommand)
		o.updateStatus("Running custom pre-process...")

		cmd := exec.Command("cmd", "/c", o.config.PreProcessCommand)
		cmd.Dir = versionDir

		if err := cmd.Run(); err != nil {
			logger.Printf("Pre-process failed: %v\n", err)
			return fmt.Errorf("pre-process failed: %v", err)
		}

		logger.Println("Custom pre-process completed")
	}

	return nil
}

// ポストプロセスを実行
func (o *OrbitApp) runPostProcess(versionDir string) error {
	if !o.config.RunPostProcess || o.config.PostProcessCommand == "" {
		return nil
	}

	logger.Printf("Running custom post-process: %s\n", o.config.PostProcessCommand)
	o.updateStatus("Running custom post-process...")

	cmd := exec.Command("cmd", "/c", o.config.PostProcessCommand)
	cmd.Dir = versionDir

	if err := cmd.Run(); err != nil {
		logger.Printf("Post-process failed: %v\n", err)
		return fmt.Errorf("post-process failed: %v", err)
	}

	logger.Println("Custom post-process completed")
	return nil
}

func (o *OrbitApp) startComfyUI(versionDir, version string) {
	logger.Printf("Starting ComfyUI %s (GPU: %s) from %s\n", version, o.config.GPUType, versionDir)

	// プレプロセスを実行
	if err := o.runPreProcess(versionDir); err != nil {
		dialog.ShowError(fmt.Errorf("Pre-process failed: %v", err), o.window)
		return
	}

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

		// batファイルを新しいコマンドプロンプトウィンドウで実行（別プロセスとして）
		// より単純なstart構文を使用
		cmd = exec.Command("cmd", "/c", "start", "/D", workDir, absPath)

		logger.Printf("Executing command: cmd /c start /D \"%s\" \"%s\"\n", workDir, absPath)
		logger.Printf("Working directory: %s\n", workDir)
		logger.Printf("Batch file path: %s\n", absPath)
		o.updateStatus(fmt.Sprintf("Starting ComfyUI from: %s", filepath.Base(absPath)))
	} else {
		// Pythonスクリプトの場合
		logger.Printf("Executing Python script: %s\n", exePath)
		cmd = exec.Command("python", exePath)
		workDir = filepath.Dir(exePath)
		cmd.Dir = workDir
	}

	logger.Printf("Starting process...\n")
	// Start()を使用して別プロセスとして起動（Wait()を呼ばない）
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

	// ポストプロセスを実行
	if err := o.runPostProcess(versionDir); err != nil {
		logger.Printf("Warning: Post-process failed: %v\n", err)
		// ポストプロセスの失敗は警告のみ（致命的ではない）
	}

	o.updateStatus(fmt.Sprintf("ComfyUI %s launched successfully! (PID: %d)", version, cmd.Process.Pid))
	logger.Printf("=== ComfyUI %s launched successfully ===\n", version)
}

func init() {
	// Windows環境でのみ動作する
	if runtime.GOOS != "windows" {
		fmt.Println("This launcher is designed for Windows only")
		os.Exit(1)
	}
}
