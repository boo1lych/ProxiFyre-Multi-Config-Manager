# ProxiFyre Configuration Manager

A modern Windows GUI application for managing [ProxiFyre](https://github.com/wiresock/proxifyre) SOCKS5 proxy configurations with built-in service control, multi-config support, and system tray integration.

![ProxiFyre Manager Screenshot](https://github.com/user-attachments/assets/36083d12-597d-4a74-9d99-e1725d915b2b)

## ✨ Features

- **🎨 Intuitive GUI** — Clean Fyne-based interface with dark/light theme support
- **🔄 Service Control** — Install, uninstall, start, stop, and restart ProxiFyre as Windows service (with UAC support)
- **📁 Multi-Config Management** — Create, clone, rename, and switch between multiple proxy configurations
- **🔌 Multiple Proxies** — Manage unlimited SOCKS5 proxy profiles per config
- **🎯 Process Exclusions** — Specify applications to bypass the proxy
- **⚡ Auto-Start on Login** — Toggle Windows autostart via system tray (no admin rights required)
- **🔍 Path Auto-Detection** — Automatically finds `ProxiFyre.exe` via current dir, PATH, or Windows Service config
- **💾 Safe Deployment** — Backs up existing config before applying changes
- **🖥️ System Tray** — Quick access menu with window show/hide, service control, and quit
- **✅ Real-time Validation** — Ensures configuration integrity before saving

---

## 📋 Prerequisites

### For Users (Running the Application)
- Windows 10/11
- [ProxiFyre](https://github.com/wiresock/proxifyre) installed
- Administrator privileges (required only for **Install/Uninstall Service** actions)

### For Developers (Building from Source)
- Go 1.20 or later
- Fyne v2 dependencies
- Windows SDK (for cross-compilation)

---

## 🚀 Installation

### Option 1: Download Pre-built Binary (Recommended)
1. Download the latest release from the [Releases](../../releases) page
2. Extract `ProxiFyreManager.exe` to your desired location
3. (Optional) Place it in the same directory as `ProxiFyre.exe` for auto-path detection
4. Run the application — no installation required

### Option 2: Build from Source

#### Windows (Native)
```cmd
# Install Go from https://golang.org/dl/
go version

# Install Fyne CLI tool
go install fyne.io/fyne/v2/cmd/fyne@latest

# Clone and build
git clone <repository-url>
cd ProxiFyre-Multi-Config-Manager
go mod tidy

# Build GUI executable (no console window)
go build -ldflags "-H windowsgui -s -w" -o ProxiFyreManager.exe .

# Or use Fyne packaging (recommended for distribution)
fyne package -os windows -appID com.proxifyre.manager -icon ProxiFyre.png -release
```

#### Cross-compile from Linux/macOS
```bash
GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui -s -w" -o ProxiFyreManager.exe .
```

---

## 🎮 Usage

### Starting the Application
1. Launch `ProxiFyreManager.exe` (double-click or from CMD)
2. The application will:
   - Auto-detect `ProxiFyre.exe` location
   - Load available configs from `configs/` directory
   - Create `configs/default.json` if none exist
3. The window can be minimized to system tray (click `✕` or use tray menu)

### Managing Configurations

| Action | How To |
|--------|--------|
| **Add Config** | Click `Add Config` → enter unique name → new empty config created |
| **Clone Config** | Select config → `Clone Config` → enter new name → copies all settings |
| **Rename Config** | Select config (except `default`) → `Rename Config` → enter new name |
| **Remove Config** | Select config → `Remove Config` → confirm (last config cannot be deleted) |
| **Switch Config** | Use dropdown in top bar → proxy list and editor update automatically |

### Managing Proxies

#### Adding a New Proxy
1. Click `Add Proxy`
2. Enter **Application Names** (one per line):
   ```
   firefox
   chrome
   C:\Program Files\WindowsApps\ROBLOX
   ```
3. Configure **SOCKS5 Endpoint**: `proxy.example.com:1080`
4. (Optional) Add credentials: username/password
5. Select protocols: `TCP` and/or `UDP`
6. Changes auto-save when switching proxies or configs

#### Editing / Removing Proxies
- Select a proxy from the list → edit fields on the right → changes save automatically
- Select proxy → `Remove Proxy` → confirm deletion

### Managing Exclusions
Applications listed here will **bypass** the proxy:
```
edge
discord
C:\Program Files\LocalApp\NotProxiedApp.exe
```
Enter one app per line in the **Global Excluded Applications** section.

### Saving & Applying Changes

| Button | Action |
|--------|--------|
| **Save Configuration** | Writes changes to `configs/<name>.json` only |
| **Save & Restart Service** 🔵 | Saves config + deploys to ProxiFyre dir + restarts service (changes take effect immediately) |
| **Reload from File** | Discards unsaved changes and reloads from disk |

> 💡 **Tip**: Use **Save & Restart Service** to apply changes without manual service restart.

### Service Management

| Button | Description | Requires Admin |
|--------|-------------|---------------|
| **Install Service** | Registers ProxiFyre as Windows service + starts it | ✅ Yes (UAC prompt) |
| **Uninstall Service** | Stops and removes the service | ✅ Yes (UAC prompt) |

> ⚠️ Service actions trigger a Windows UAC prompt. Click **Yes** to proceed.

### System Tray Menu
Right-click the tray icon (near clock) for quick actions:
- `Show Window` — Restore the main window
- `☐/✓ Launch on Startup` — Toggle Windows autostart (HKCU registry, no admin needed)
- `Install/Uninstall Service` — Quick access to service management
- `Quit` — Exit the application

---

## ⚙️ Configuration File Format

Configs are stored in `configs/<name>.json`:

```json
{
  "logLevel": "Error",
  "proxifyrePath": "C:\\ProxiFyre\\ProxiFyre.exe",
  "proxies": [
    {
      "appNames": ["chrome", "firefox"],
      "socks5ProxyEndpoint": "proxy.example.com:1080",
      "username": "myuser",
      "password": "mypass",
      "supportedProtocols": ["TCP", "UDP"]
    }
  ],
  "excludes": ["edge", "localservice.exe"]
}
```

### Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `logLevel` | string | Logging verbosity: `Error`, `Warning`, `Info`, `Debug`, `All` |
| `proxifyrePath` | string | (Optional) Full path to `ProxiFyre.exe` |
| `proxies` | array | List of proxy configurations |
| `appNames` | array | Application names/paths to route through proxy |
| `socks5ProxyEndpoint` | string | Proxy server address and port |
| `username` / `password` | string | Optional SOCKS5 authentication |
| `supportedProtocols` | array | `TCP`, `UDP`, or both |
| `excludes` | array | Applications to **bypass** the proxy |

---

## 🔧 Advanced Usage

### ProxiFyre Path Management
If auto-detection fails:
1. Enter the full path to `ProxiFyre.exe` in the **ProxiFyre Path** field
2. Click `Apply` to save
3. Or click `Detect Path` to re-run auto-detection

### Deployment Behavior
When using **Save & Restart Service**:
1. Current config is saved to `configs/<name>.json`
2. Config is copied to ProxiFyre directory as `app-config.json` (with backup)
3. Service is restarted to apply changes

### Logging
Application logs to console during development. For production builds (`-H windowsgui`), use:
```go
log.Printf("[Service] Your message here")
```
Logs appear in Windows Event Viewer or can be redirected to file.

### Autostart Mechanism
- Uses `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` (current user only)
- No admin rights required
- Toggle via system tray menu → changes apply immediately

---

## 🛠️ Troubleshooting

| Issue | Solution |
|-------|----------|
| **App won't start** | Ensure `ProxiFyre.png` is in the same directory; run from CMD to see errors |
| **Service install fails** | Click **Yes** in UAC prompt; verify you have admin rights |
| **Config not applying** | Use **Save & Restart Service**, not just Save; check service status via `sc query ProxiFyre` |
| **Proxy not working for app** | Verify app name/path; check exclusions list; ensure protocol (TCP/UDP) matches app traffic |
| **Path detection fails** | Manually enter full path to `ProxiFyre.exe` and click Apply |
| **Tray icon missing** | Ensure app is built with Fyne desktop driver; check Windows notification area settings |

---

## 📦 Distribution

### Using `build.bat` (Windows)
A ready-to-use build script is included:
```cmd
build.bat
```
Creates `dist/` folder with:
- `ProxiFyreManager.exe` (GUI build, no console)
- `ProxiFyre.png` (icon)
- `configs/default.json` (starter config)
- `README.txt` + `version.txt`

### Manual Packaging
```cmd
mkdir release
copy ProxiFyreManager.exe release\
copy ProxiFyre.png release\
copy README.md release\
```

### Code Signing (Optional)
For enterprise deployment, sign the executable:
```cmd
signtool sign /fd SHA256 /a /tr http://timestamp.digicert.com /td SHA256 ProxiFyreManager.exe
```

---

## 🧑‍💻 Development

### Project Structure
```
ProxiFyre-Multi-Config-Manager/
├── main.go              # Main application code
├── go.mod / go.sum      # Go module dependencies
├── ProxiFyre.png        # Application icon
├── build.bat            # Windows build script
├── configs/             # Runtime config storage (gitignored)
└── README.md            # This file
```

### Key Components
| File/Function | Purpose |
|---------------|---------|
| `ConfigManager` | Manages multiple configs with load/save/switch logic |
| `updateSystemTrayMenu()` | Dynamically builds tray menu with autostart status |
| `runElevated()` | Executes commands with UAC prompt via PowerShell |
| `DeployConfigToProxiFyre()` | Safely copies config to ProxiFyre directory with backup |
| `resolveProxiFyrePath()` | Multi-method path detection (dir, PATH, service config) |

### Adding New Features
1. Follow existing Fyne widget patterns
2. Use `dialog.Show*` for user feedback
3. Log service actions with `log.Printf("[Service] ...")`
4. Test UAC flows with non-admin user account

---

## 🤝 Contributing
Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push and open a Pull Request

### Code Guidelines
- Use `fmt.Errorf` with `%w` for error wrapping
- Keep UI logic in `main()`; extract helpers to functions
- Test service actions on clean Windows VM
- Update this README for user-facing changes

---

## 📜 License
MIT License — see [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments
- Built with [Fyne](https://fyne.io/) — Cross-platform Go GUI toolkit
- Created for [ProxiFyre](https://github.com/wiresock/proxifyre) by Wiresock
- Icons and design inspired by modern Windows applications

---

## 🆘 Support
| Issue Type | Where to Report |
|------------|----------------|
| GUI application bugs | [This repository → Issues](../../issues) |
| ProxiFyre core issues | [ProxiFyre repository](https://github.com/wiresock/proxifyre) |
| Feature requests | [Discussions tab](../../discussions) |

> ℹ️ **Note**: This is an unofficial GUI tool for ProxiFyre. For official ProxiFyre documentation and support, visit the [official repository](https://github.com/wiresock/proxifyre).

---

## 🔄 Changelog

### v2.0.0 (Current)
- ✨ **Multi-config support**: Add, clone, rename, remove configs via UI
- 🚀 **Autostart toggle**: Enable/disable Windows login launch from system tray
- 🔐 **UAC integration**: Proper admin elevation for service install/uninstall
- 🎨 **UI polish**: Highlighted `Save & Restart` button, auto-select first proxy on config switch
- 🔍 **Smart path detection**: Finds ProxiFyre via dir, PATH, or Windows Service config
- 💾 **Safe deployment**: Backs up `app-config.json` before overwriting
- 🖥️ **System tray menu**: Dynamic menu with live autostart status
- 🛠️ **Build script**: `build.bat` for one-click release packaging

### v1.0.0 (Initial Release)
- ✨ Full configuration management GUI
- 🔄 Service restart integration
- 📝 Multiple proxy support per config
- 🎯 Process exclusion management
- 💾 Real-time configuration validation
- 🔐 Secure password handling

---

*Last updated: April 2026*  
*Compatible with ProxiFyre v1.0+*