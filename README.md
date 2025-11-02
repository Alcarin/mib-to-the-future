# MIB to the Future

<p align="center">
  <img src="frontend/src/assets/images/delorean.png" alt="MIB to the Future Logo" width="250"/>
</p>

<p align="center">
  A modern, fast, and intuitive SNMP MIB browser, built to explore the future of network management.
</p>


<p align="center">
  <a href="https://creativecommons.org/licenses/by-nc/4.0/"><img src="https://img.shields.io/badge/License-CC%20BY--NC%204.0-lightgrey.svg" alt="License: CC BY-NC 4.0"></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="https://wails.io/"><img src="https://img.shields.io/badge/Wails-DF0000?logo=wails&logoColor=white" alt="Wails"></a>
  <a href="https://vuejs.org/"><img src="https://img.shields.io/badge/Vue.js-4FC08D?logo=vue.js&logoColor=white" alt="Vue.js"></a>
  <a href="https://vitejs.dev/"><img src="https://img.shields.io/badge/Vite-646CFF?logo=vite&logoColor=white" alt="Vite"></a>
  <a href="https://material-web.dev/"><img src="https://img.shields.io/badge/Material%20Design%203-757575?logo=material-design&logoColor=white" alt="Material Design"></a>
  <a href="https://www.sqlite.org/"><img src="https://img.shields.io/badge/SQLite-003B57?logo=sqlite&logoColor=white" alt="SQLite"></a>
</p>

---

**MIB to the Future** is a hybrid desktop application for monitoring and managing network devices using SNMP. It leverages the power of **Go** for the backend and the elegance of **Vue.js** with **Material 3** for a responsive and modern user interface, all brought together by **Wails**.

The project aims to provide a comprehensive tool for both network novices and professionals, combining advanced features with a simple and enjoyable user experience.

## Features

### 🚀 SNMP Protocol Support
- **Complete SNMP Operations**: Execute GET, GET-NEXT, WALK, and SET requests with an intuitive interface
- **SNMPv1, v2c, and v3**: Full protocol support including SNMPv3 authentication and encryption for secure device management
- **Multi-Device Management**: Save unlimited connection profiles and switch instantly between network devices

### 📚 Smart MIB Browser
- **Persistent MIB Database**: Import and parse custom MIB files into a high-performance SQLite database
- **Interactive Tree Navigation**: Explore MIB hierarchies with an expandable, searchable tree view
- **Instant OID Search**: Find any OID in milliseconds with real-time search across your entire MIB collection
- **Comprehensive OID Details**: View syntax, access rights, descriptions, and metadata for every MIB object
- **Bookmark System**: Save frequently used OIDs and organize them in customizable folders with drag-and-drop support

### 📊 Advanced Data Visualization
- **Automatic Table Rendering**: SNMP table data displayed in clean, sortable table views
- **Real-Time Graphing**: Visualize numerical OID values over time with interactive charts
- **Column Operations**: Perform bulk operations on entire SNMP table columns
- **Multi-Format Export**: Export results to CSV for further analysis

### 💼 Professional Workspace
- **Tabbed Interface**: Organize multiple queries in renameable tabs for efficient multitasking
- **Smart Results Log**: Filter by status (Success, Error, Pending) and search through results with full-text search
- **Material Design 3**: Modern, responsive UI with customizable color themes
- **Dark Mode Support**: Comfortable viewing in any lighting condition
- **Session Persistence**: Your workspace, connections, and bookmarks are automatically saved

## Roadmap (Planned Features)


## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [Releases page](https://github.com/Alcarin/mib-to-the-future/releases):

- **Linux**: `mib-to-the-future-linux-amd64` (x86-64)
- **Windows**: `mib-to-the-future-windows-amd64.exe` (x86-64)
- **macOS**: `mib-to-the-future-darwin-amd64.app.zip` (Intel) or `mib-to-the-future-darwin-arm64.app.zip` (Apple Silicon)

#### Linux Installation

1. Download the binary
2. Make it executable:
   ```bash
   chmod +x mib-to-the-future-linux-amd64
   ```
3. Run it:
   ```bash
   ./mib-to-the-future-linux-amd64
   ```

**Required dependencies** (Ubuntu/Debian):
```bash
sudo apt-get install libgtk-3-0 libwebkit2gtk-4.1-0
```

#### Windows Installation

1. Download the `.exe` file
2. Double-click to run

**Security Note**: Since the binaries are unsigned, Windows SmartScreen may show a warning:
- Click "More info" → "Run anyway"
- Or right-click the file → Properties → check "Unblock" → Apply → OK

#### macOS Installation

1. Download and unzip the `.app.zip` file for your Mac:
   - Intel Macs: `darwin-amd64`
   - Apple Silicon (M1/M2/M3): `darwin-arm64`
2. Move the app to your Applications folder
3. **First launch**: Since the app is unsigned, macOS Gatekeeper will block it. Choose one method:

   **Method 1 - Right-click bypass (easiest)**:
   - Right-click (or Control+click) on the app
   - Select "Open" from the menu
   - Click "Open" in the dialog

   **Method 2 - System Settings**:
   - Try to open the app normally (it will be blocked)
   - Go to System Settings → Privacy & Security
   - Scroll down to find "MIB to the Future was blocked"
   - Click "Open Anyway"

   **Method 3 - Terminal (advanced)**:
   ```bash
   xattr -cr /Applications/mib-to-the-future.app
   ```

**Why unsigned?**: Code signing requires paid developer certificates ($99/year for Apple, certificate costs for Windows). This is an open-source project without commercial funding. The source code is fully available for audit, and you can build from source if preferred.

### Build from Source

#### Prerequisites

Before building from source, you need to install the following tools:

1. **Go** (version 1.21 or higher)
   - **Linux/macOS**: Download from [golang.org](https://go.dev/dl/) or use your package manager
     ```bash
     # Ubuntu/Debian
     sudo apt-get install golang-go

     # macOS with Homebrew
     brew install go
     ```
   - **Windows**: Download installer from [golang.org](https://go.dev/dl/)

2. **Node.js and npm** (version 16 or higher)
   - **Linux/macOS**: Download from [nodejs.org](https://nodejs.org/) or use your package manager
     ```bash
     # Ubuntu/Debian
     sudo apt-get install nodejs npm

     # macOS with Homebrew
     brew install node
     ```
   - **Windows**: Download installer from [nodejs.org](https://nodejs.org/)

3. **Wails CLI** (version 2.10.2)
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.2
   ```

4. **Platform-specific dependencies**:
   - **Linux** (Ubuntu/Debian):
     ```bash
     sudo apt-get install libgtk-3-dev libwebkit2gtk-4.1-dev
     ```
   - **macOS**: Xcode Command Line Tools
     ```bash
     xcode-select --install
     ```
   - **Windows**: No additional dependencies required

#### Building the Application

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/Alcarin/mib-to-the-future.git
    cd mib-to-the-future
    ```

2.  **Install frontend dependencies:**
    ```bash
    cd frontend
    npm install
    cd ..
    ```

3.  **Run in development mode:**
    ```bash
    wails dev
    ```

4.  **Build for production:**
    ```bash
    # Build for your current platform
    wails build

    # Build for Linux with WebKit 4.1 support (Ubuntu 24.04+)
    wails build -tags webkit2_41

    # Cross-compile for other platforms
    wails build -platform linux/amd64
    wails build -platform windows/amd64
    wails build -platform darwin/arm64
    ```

The compiled binary will be in the `build/bin` directory.

## Contributing

Contributions are welcome! Please feel free to open an issue to discuss a new feature or submit a pull request.
