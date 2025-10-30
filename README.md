# Orbit - ComfyUI Launcher

ComfyUI portable版を簡単に管理・起動するためのランチャーアプリケーションです。

## 特徴

- 🚀 **バージョン管理**: ComfyUIの複数バージョンを管理・切り替え
- 📦 **自動ダウンロード**: GitHubから最新リリースを自動取得
- 💾 **キャッシュ機能**: 一度ダウンロードしたバージョンは再利用
- 🎨 **モダンなUI**: 直感的でスタイリッシュなインターフェース
- ⚡ **高速起動**: バージョンを選択するだけで即座に起動
- 🎮 **GPU自動検出**: NVIDIA/AMD/CPUを自動判別して最適なパッケージをダウンロード
- 🔧 **カスタムアプリ起動**: 6個まで任意のアプリケーションを登録可能
- 📊 **詳細ログ**: すべての操作をLogsディレクトリに記録

## 必要要件

- Windows 10/11 (64-bit)
- [Go 1.25.3+](https://golang.org/dl/)
- [7-Zip](https://www.7-zip.org/) (自動解凍に必要)
- Git (開発環境でインストール済み前提)

## インストール

1. リポジトリをクローン:
```bash
git clone <repository-url>
cd Orbit
```

2. 依存関係をインストール:
```bash
go mod tidy
```

3. ビルド:
```bash
build.bat
```

または手動でビルド:
```bash
go build -ldflags="-H windowsgui" -o orbit.exe main.go
```

## 使い方

### 基本的な使い方

1. **`orbit.exe` を実行**
2. **GPU Typeを確認/選択**
   - 自動検出されたGPUが表示されます
   - 必要に応じて手動で変更可能（NVIDIA GPU / AMD GPU / CPU）
3. **「Refresh Versions」ボタン**でComfyUIのリリース一覧を取得
4. **バージョンを選択**
   - インストール状態が表示されます（✓ Installed / ✗ Not Installed）
5. **「Launch ComfyUI」ボタン**で起動

### 初回起動時

- 選択したバージョンが自動的に`temp/`にダウンロードされます
- `packages/<バージョン名>/` に展開されます
- GPUタイプに応じた正しいパッケージがダウンロードされます
  - NVIDIA: `ComfyUI_windows_portable_nvidia.7z`
  - AMD: `ComfyUI_windows_portable_amd.7z`
  - CPU: `ComfyUI_windows_portable_cpu.7z`
- 解凍完了後、自動的に起動します

### 2回目以降

- 既にダウンロード済みのバージョンは即座に起動します
- ダウンロード処理はスキップされます

### カスタムアプリの設定

1. 各アプリの **⚙ ボタン**をクリック
2. アプリ名とパスを設定
3. アプリボタンをクリックで起動

### ログの確認

- すべての操作は`Logs/`ディレクトリに記録されます
- ログファイル名: `orbit_YYYY-MM-DD_HH-MM-SS.log`
- トラブルシューティングに活用できます

## プロジェクト構造

```
Orbit/
├── main.go              # メインアプリケーション
├── go.mod               # Go モジュール定義
├── go.sum               # Go 依存関係チェックサム
├── build.bat            # ビルドスクリプト
├── CLAUDE.md            # プロジェクト仕様書
├── README.md            # このファイル
├── LICENSE              # ライセンス
├── .gitignore           # Git除外設定
├── Img/
│   └── banner.png       # アプリケーションロゴ
├── Misc/
│   └── mingw64/         # MinGW-w64コンパイラ（ビルド用）
├── orbit.exe            # ビルド後の実行ファイル（.gitignoreで除外）
├── orbit_config.json    # 設定ファイル（自動生成、.gitignoreで除外）
├── packages/            # ダウンロードしたComfyUI（.gitignoreで除外）
│   ├── v0.3.67/
│   ├── v0.3.66/
│   └── ...
├── temp/                # 一時ダウンロード（.gitignoreで除外）
└── Logs/                # ログファイル（.gitignoreで除外）
    ├── orbit_2025-10-30_18-11-05.log
    └── ...
```

## 技術仕様

- **言語**: Go 1.25.3
- **UIフレームワーク**: Fyne v2.5.3
- **GitHub API**: ComfyUIのリリース情報を動的取得
- **圧縮形式**: 7z (7-Zipで解凍)
- **ログ**: タイムスタンプ付きファイル + 標準出力

## 実装済み機能

### ✅ コア機能

1. **動的バージョン取得**: GitHub Releases APIからリアルタイムで取得
2. **GPU自動検出**: nvidia-smi / wmicでGPUを自動判別
3. **GPU別パッケージ管理**: NVIDIA/AMD/CPU用パッケージを自動選択
4. **キャッシュ機構**: ダウンロード済みバージョンの自動判定
5. **設定の永続化**: 最後に使用したバージョンとGPUタイプを記憶

### ✅ UI機能

6. **モダンなインターフェース**: ロゴ、バージョン選択、GPU選択を統合
7. **インストール状態表示**: バージョン・GPUタイプごとの状態を可視化
8. **カスタムアプリランチャー**: 6個まで任意のアプリを登録・起動
9. **進捗表示**: ダウンロード・解凍時のプログレスダイアログ
10. **エラーハンドリング**: 適切なエラーメッセージとログ記録

### ✅ 開発者向け

11. **詳細ログ**: すべての操作をタイムスタンプ付きで記録
12. **デバッグ対応**: コマンドプロンプトウィンドウでエラー確認可能

## 🔮 今後の改善案

1. **ダウンロード進捗バー**: バイト数・速度表示
2. **バージョン削除機能**: UI上から不要なバージョンを削除
3. **自動更新チェック**: 新しいComfyUIバージョンの通知
4. **カスタムパス設定**: インストール先を変更可能に
5. **多言語対応**: 英語・日本語切り替え
6. **起動オプション設定**: メモリ制限、ポート番号などのカスタマイズ
7. **テーマ切り替え**: ライト/ダークモード

## トラブルシューティング

### 7-Zipが見つからない場合
```
Error: 7-Zip not found
```
→ [7-Zip](https://www.7-zip.org/)をインストールしてください

### ダウンロードが失敗する場合
- インターネット接続を確認
- ファイアウォールの設定を確認
- GitHub APIのレート制限の可能性（1時間待ってから再試行）

### ComfyUIが起動しない場合
- NVIDIA GPU ドライバーが最新か確認
- Python環境が正しくインストールされているか確認
- `packages/<バージョン>/ComfyUI_windows_portable/` の内容を確認

## ライセンス

MIT License

## 開発者向け

### デバッグビルド
```bash
go build -o orbit.exe main.go
```

### リリースビルド（ウィンドウなし）
```bash
go build -ldflags="-H windowsgui" -o orbit.exe main.go
```

### 依存関係の更新
```bash
go get -u fyne.io/fyne/v2@latest
go mod tidy
```

## 貢献

プルリクエストや問題報告は歓迎します！

## 謝辞

- [ComfyUI](https://github.com/comfyanonymous/ComfyUI) - 素晴らしいツールを提供してくださっているチームに感謝
- [Fyne](https://fyne.io/) - クロスプラットフォームGUIフレームワーク
