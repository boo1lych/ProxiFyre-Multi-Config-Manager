package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/registry"
)

const autoRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const autoRunVal = `ProxiFyreManager`

func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoRunKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(autoRunVal)
	return err == nil
}

func toggleAutoStart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoRunKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()
	exe, _ := os.Executable()
	if isAutoStartEnabled() {
		return k.DeleteValue(autoRunVal)
	}
	return k.SetStringValue(autoRunVal, exe)
}

func updateSystemTrayMenu(desk desktop.App, win fyne.Window, app fyne.App) {
	startupLabel := "☐ Launch on Startup"
	if isAutoStartEnabled() {
		startupLabel = "✓ Launch on Startup"
	}

	m := fyne.NewMenu("ProxiFyre Manager",
		fyne.NewMenuItem("Show Window", func() { win.Show() }),
		fyne.NewMenuItem(startupLabel, func() {
			if err := toggleAutoStart(); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to toggle autostart: %v", err), win)
			}
			updateSystemTrayMenu(desk, win, app) // Рекурсивно обновляем меню
		}),
		fyne.NewMenuItem("Install Service", func() {
			win.Show()
			dialog.ShowConfirm("Install as Service", "This will install ProxiFyre as a service. Continue?", func(confirmed bool) {
				if confirmed {
					if err := installService(); err != nil {
						dialog.ShowError(fmt.Errorf("Failed to install service: %v", err), win)
					} else {
						dialog.ShowInformation("Success", "Service installed successfully!", win)
					}
				}
			}, win)
		}),
		fyne.NewMenuItem("Uninstall Service", func() {
			win.Show()
			dialog.ShowConfirm("Uninstall as Service", "This will uninstall ProxiFyre as a service. Continue?", func(confirmed bool) {
				if confirmed {
					if err := uninstallService(); err != nil {
						dialog.ShowError(fmt.Errorf("Failed to uninstall service: %v", err), win)
					} else {
						dialog.ShowInformation("Success", "Service uninstalled successfully!", win)
					}
				}
			}, win)
		}),
		fyne.NewMenuItem("Quit", func() {
			win.Show()
			dialog.ShowConfirm("Quit Manager", "This will quit ProxiFyre Manager. Continue?", func(confirmed bool) {
				if confirmed {
					app.Quit()
				}
			}, win)
		}),
	)
	desk.SetSystemTrayMenu(m)
}

// Config structures matching ProxiFyre configuration
type ProxyConfig struct {
	AppNames            []string `json:"appNames"`
	Socks5ProxyEndpoint string   `json:"socks5ProxyEndpoint"`
	Username            string   `json:"username,omitempty"`
	Password            string   `json:"password,omitempty"`
	SupportedProtocols  []string `json:"supportedProtocols"`
}

type AppConfig struct {
	LogLevel      string        `json:"logLevel"`
	ProxifyrePath string        `json:"proxifyrePath,omitempty"`
	Proxies       []ProxyConfig `json:"proxies"`
	Excludes      []string      `json:"excludes"`
}

type ConfigEntry struct {
	Name string
	Path string
}

type ConfigManager struct {
	CurrentConfig      AppConfig
	Configs            []ConfigEntry
	ProxifyrePath      string
	CurrentConfigIndex int
}

var configManager ConfigManager

func (cm *ConfigManager) LoadConfigs() {
	cm.Configs = []ConfigEntry{}
	if err := os.MkdirAll("configs", 0755); err != nil {
		return
	}
	files, err := os.ReadDir("configs")
	if err != nil {
		return
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			name := strings.TrimSuffix(file.Name(), ".json")
			cm.Configs = append(cm.Configs, ConfigEntry{Name: name, Path: filepath.Join("configs", file.Name())})
		}
	}
	if len(cm.Configs) == 0 {
		// Create default
		cm.Configs = append(cm.Configs, ConfigEntry{Name: "default", Path: "configs/default.json"})
		cm.CurrentConfigIndex = 0
		cm.SaveCurrentConfig()
	} else {
		cm.CurrentConfigIndex = 0
		cm.LoadCurrentConfig()
	}
}

func (cm *ConfigManager) LoadCurrentConfig() {
	if cm.CurrentConfigIndex < 0 || cm.CurrentConfigIndex >= len(cm.Configs) {
		return
	}
	path := cm.Configs[cm.CurrentConfigIndex].Path
	data, err := os.ReadFile(path)
	if err != nil {
		cm.CurrentConfig = AppConfig{LogLevel: "Error", Proxies: []ProxyConfig{}, Excludes: []string{}}
		return
	}
	if err := json.Unmarshal(data, &cm.CurrentConfig); err != nil {
		cm.CurrentConfig = AppConfig{LogLevel: "Error", Proxies: []ProxyConfig{}, Excludes: []string{}}
	}
	if cm.CurrentConfig.ProxifyrePath != "" {
		cm.ProxifyrePath = cm.CurrentConfig.ProxifyrePath
	}
}

func (cm *ConfigManager) SaveCurrentConfig() {
	if cm.CurrentConfigIndex < 0 || cm.CurrentConfigIndex >= len(cm.Configs) {
		return
	}
	path := cm.Configs[cm.CurrentConfigIndex].Path
	data, err := json.MarshalIndent(cm.CurrentConfig, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(path, data, 0644)
}

func (cm *ConfigManager) DetectProxifyrePath() {
	// Try current dir
	if _, err := os.Stat("ProxiFyre.exe"); err == nil {
		cm.ProxifyrePath = "ProxiFyre.exe"
		cm.CurrentConfig.ProxifyrePath = cm.ProxifyrePath
		return
	}
	// Try exe dir
	exePath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exePath)
		proxifyrePath := filepath.Join(dir, "ProxiFyre.exe")
		if _, err := os.Stat(proxifyrePath); err == nil {
			cm.ProxifyrePath = proxifyrePath
			cm.CurrentConfig.ProxifyrePath = cm.ProxifyrePath
			return
		}
	}
	// Try PATH
	path, err := exec.LookPath("ProxiFyre.exe")
	if err == nil {
		cm.ProxifyrePath = path
		cm.CurrentConfig.ProxifyrePath = cm.ProxifyrePath
		return
	}

	// Try Windows service config
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("sc", "qc", "ProxiFyre").CombinedOutput(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "BINARY_PATH_NAME") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						binaryPath := strings.TrimSpace(parts[1])
						binaryPath = strings.Trim(binaryPath, "\"")
						if binaryPath != "" {
							cm.ProxifyrePath = binaryPath
							cm.CurrentConfig.ProxifyrePath = cm.ProxifyrePath
							return
						}
					}
				}
			}
		}
	}
}

func (cm *ConfigManager) SetProxifyrePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, "ProxiFyre.exe")
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("ProxiFyre.exe not found at %s", path)
	}

	cm.ProxifyrePath = path
	cm.CurrentConfig.ProxifyrePath = path
	return nil
}

func main() {
	fmt.Println("Starting ProxiFyre Configuration Manager")
	myApp := app.New()
	myWindow := myApp.NewWindow("ProxiFyre Configuration Manager")
	myWindow.Resize(fyne.NewSize(900, 700))

	if iconResource, err := fyne.LoadResourceFromPath("ProxiFyre.png"); err == nil {
		myApp.SetIcon(iconResource)
		myWindow.SetIcon(iconResource)
	}

	if desk, ok := myApp.(desktop.App); ok {
		updateSystemTrayMenu(desk, myWindow, myApp)
	}

	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})

	configManager.LoadConfigs()
	configManager.DetectProxifyrePath()

	// Create UI elements
	logLevelSelect := widget.NewSelect(
		[]string{"Error", "Warning", "Info", "Debug", "All"},
		func(value string) {
			configManager.CurrentConfig.LogLevel = value
		},
	)
	logLevelSelect.SetSelected(configManager.CurrentConfig.LogLevel)

	// Proxy list
	proxyList := widget.NewList(
		func() int { return len(configManager.CurrentConfig.Proxies) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Proxy")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(fmt.Sprintf("Proxy %d: %s", id+1, configManager.CurrentConfig.Proxies[id].Socks5ProxyEndpoint))
		},
	)

	var selectedProxyIndex int = -1
	var proxyEditor *fyne.Container

	// Proxy editor
	appNamesEntry := widget.NewMultiLineEntry()
	appNamesEntry.SetPlaceHolder("One app per line (e.g., firefox, chrome)")
	appNamesEntry.Wrapping = fyne.TextWrapWord
	appNamesEntry.SetMinRowsVisible(3)

	endpointEntry := widget.NewEntry()
	endpointEntry.SetPlaceHolder("proxy.example.com:1080")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Optional username")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Optional password")

	tcpCheck := widget.NewCheck("TCP", nil)
	udpCheck := widget.NewCheck("UDP", nil)

	excludesEntry := widget.NewMultiLineEntry()
	excludesEntry.SetPlaceHolder("One app per line to exclude from proxy")
	excludesEntry.Wrapping = fyne.TextWrapWord
	excludesEntry.SetMinRowsVisible(3)
	if len(configManager.CurrentConfig.Excludes) > 0 {
		excludesEntry.SetText(strings.Join(configManager.CurrentConfig.Excludes, "\n"))
	}

	updateProxyEditor := func(index int) {
		if index < 0 || index >= len(configManager.CurrentConfig.Proxies) {
			appNamesEntry.SetText("")
			endpointEntry.SetText("")
			usernameEntry.SetText("")
			passwordEntry.SetText("")
			tcpCheck.SetChecked(false)
			udpCheck.SetChecked(false)
			return
		}

		proxy := configManager.CurrentConfig.Proxies[index]
		appNamesEntry.SetText(strings.Join(proxy.AppNames, "\n"))
		endpointEntry.SetText(proxy.Socks5ProxyEndpoint)
		usernameEntry.SetText(proxy.Username)
		passwordEntry.SetText(proxy.Password)

		tcpCheck.SetChecked(false)
		udpCheck.SetChecked(false)
		for _, proto := range proxy.SupportedProtocols {
			if strings.ToUpper(proto) == "TCP" {
				tcpCheck.SetChecked(true)
			}
			if strings.ToUpper(proto) == "UDP" {
				udpCheck.SetChecked(true)
			}
		}
	}

	saveProxyChanges := func() {
		if selectedProxyIndex < 0 || selectedProxyIndex >= len(configManager.CurrentConfig.Proxies) {
			return
		}

		appNames := strings.Split(appNamesEntry.Text, "\n")
		var cleanedAppNames []string
		for _, name := range appNames {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				cleanedAppNames = append(cleanedAppNames, trimmed)
			}
		}

		var protocols []string
		if tcpCheck.Checked {
			protocols = append(protocols, "TCP")
		}
		if udpCheck.Checked {
			protocols = append(protocols, "UDP")
		}

		configManager.CurrentConfig.Proxies[selectedProxyIndex] = ProxyConfig{
			AppNames:            cleanedAppNames,
			Socks5ProxyEndpoint: endpointEntry.Text,
			Username:            usernameEntry.Text,
			Password:            passwordEntry.Text,
			SupportedProtocols:  protocols,
		}

		proxyList.Refresh()
	}

	proxyList.OnSelected = func(id widget.ListItemID) {
		saveProxyChanges()
		selectedProxyIndex = id
		updateProxyEditor(id)
	}

	// Buttons
	addProxyBtn := widget.NewButton("Add Proxy", func() {
		saveProxyChanges()
		newProxy := ProxyConfig{
			AppNames:            []string{},
			Socks5ProxyEndpoint: "",
			SupportedProtocols:  []string{"TCP"},
		}
		configManager.CurrentConfig.Proxies = append(configManager.CurrentConfig.Proxies, newProxy)
		selectedProxyIndex = len(configManager.CurrentConfig.Proxies) - 1
		proxyList.Refresh()
		proxyList.Select(selectedProxyIndex)
	})

	removeProxyBtn := widget.NewButton("Remove Proxy", func() {
		if selectedProxyIndex < 0 || selectedProxyIndex >= len(configManager.CurrentConfig.Proxies) {
			dialog.ShowInformation("Error", "Please select a proxy to remove", myWindow)
			return
		}

		dialog.ShowConfirm("Confirm Delete", "Are you sure you want to delete this proxy?", func(confirmed bool) {
			if confirmed {
				configManager.CurrentConfig.Proxies = append(configManager.CurrentConfig.Proxies[:selectedProxyIndex], configManager.CurrentConfig.Proxies[selectedProxyIndex+1:]...)
				selectedProxyIndex = -1
				proxyList.Refresh()
				updateProxyEditor(-1)
			}
		}, myWindow)
	})

	saveBtn := widget.NewButton("Save Configuration", func() {
		saveProxyChanges()

		// Update excludes
		excludeLines := strings.Split(excludesEntry.Text, "\n")
		var cleanedExcludes []string
		for _, line := range excludeLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				cleanedExcludes = append(cleanedExcludes, trimmed)
			}
		}
		configManager.CurrentConfig.Excludes = cleanedExcludes

		configManager.SaveCurrentConfig()

		dialog.ShowInformation("Success", "Configuration saved successfully!", myWindow)
	})

	saveAndRestartBtn := widget.NewButton("Save & Restart Service", func() {
		saveProxyChanges()

		// Update excludes
		excludeLines := strings.Split(excludesEntry.Text, "\n")
		var cleanedExcludes []string
		for _, line := range excludeLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				cleanedExcludes = append(cleanedExcludes, trimmed)
			}
		}
		configManager.CurrentConfig.Excludes = cleanedExcludes

		configManager.SaveCurrentConfig()

		// Deploy config to ProxiFyre directory
		if err := configManager.DeployConfigToProxiFyre(); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to deploy config: %v", err), myWindow)
			return
		}

		dialog.ShowConfirm("Restart Service", "Configuration saved and deployed. Restart ProxiFyre service now?", func(confirmed bool) {
			if confirmed {
				if err := restartService(); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to restart service: %v", err), myWindow)
				} else {
					dialog.ShowInformation("Success", "Service restarted successfully!", myWindow)
				}
			}
		}, myWindow)
	})

	saveAndRestartBtn.Importance = widget.HighImportance

	exitBtn := widget.NewButton("Exit Manager", func() {
		dialog.ShowConfirm("Confirm Exit", "Are you sure you want to exit this manager?", func(confirmed bool) {
			if confirmed {
				myApp.Quit()
			}
		}, myWindow)
	})

	loadBtn := widget.NewButton("Reload from File", func() {
		dialog.ShowConfirm("Reload Configuration", "This will discard unsaved changes. Continue?", func(confirmed bool) {
			if confirmed {
				configManager.LoadCurrentConfig()
				logLevelSelect.SetSelected(configManager.CurrentConfig.LogLevel)
				selectedProxyIndex = -1
				proxyList.Refresh()
				updateProxyEditor(-1)
				if len(configManager.CurrentConfig.Excludes) > 0 {
					excludesEntry.SetText(strings.Join(configManager.CurrentConfig.Excludes, "\n"))
				} else {
					excludesEntry.SetText("")
				}
				dialog.ShowInformation("Success", "Configuration reloaded!", myWindow)
			}
		}, myWindow)
	})

	// Layout
	proxyEditor = container.NewVBox(
		widget.NewLabel("Proxy Configuration"),
		widget.NewForm(
			widget.NewFormItem("Application Names", appNamesEntry),
			widget.NewFormItem("SOCKS5 Endpoint", endpointEntry),
			widget.NewFormItem("Username", usernameEntry),
			widget.NewFormItem("Password", passwordEntry),
		),
		widget.NewLabel("Supported Protocols:"),
		container.NewHBox(tcpCheck, udpCheck),
	)

	leftPanel := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Proxy List"),
			container.NewHBox(addProxyBtn, removeProxyBtn),
		),
		nil,
		nil,
		nil,
		proxyList,
	)

	rightPanel := container.NewVBox(
		proxyEditor,
		widget.NewSeparator(),
		widget.NewLabel("Global Excluded Applications"),
		excludesEntry,
	)

	split := container.NewHSplit(leftPanel, container.NewScroll(rightPanel))
	split.SetOffset(0.3)

	// Config selector
	var configNames []string
	for _, c := range configManager.Configs {
		configNames = append(configNames, c.Name)
	}
	configSelector := widget.NewSelect(configNames, func(value string) {
		for i, c := range configManager.Configs {
			if c.Name == value {
				configManager.CurrentConfigIndex = i
				configManager.LoadCurrentConfig()
				logLevelSelect.SetSelected(configManager.CurrentConfig.LogLevel)

				if len(configManager.CurrentConfig.Proxies) > 0 {
					selectedProxyIndex = 0
					updateProxyEditor(0)
				} else {
					selectedProxyIndex = -1
					updateProxyEditor(-1)
				}
				proxyList.Refresh()
				if len(configManager.CurrentConfig.Excludes) > 0 {
					excludesEntry.SetText(strings.Join(configManager.CurrentConfig.Excludes, "\n"))
				} else {
					excludesEntry.SetText("")
				}
				break
			}
		}
	})
	configSelector.SetSelected(configManager.Configs[configManager.CurrentConfigIndex].Name)

	// Helper function to refresh config selector
	refreshConfigSelector := func() {
		var newNames []string
		for _, c := range configManager.Configs {
			newNames = append(newNames, c.Name)
		}
		configNames = newNames
		configSelector.Options = configNames
		configSelector.Refresh()
	}

	addConfigBtn := widget.NewButton("Add Config", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Config name")
		content := container.NewVBox(widget.NewLabel("Enter config name:"), nameEntry)
		dialog.ShowCustomConfirm("Add Config", "Create", "Cancel", content, func(ok bool) {
			if !ok {
				return
			}
			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				dialog.ShowInformation("Error", "Config name cannot be empty", myWindow)
				return
			}
			for _, c := range configManager.Configs {
				if c.Name == name {
					dialog.ShowInformation("Error", "Config with this name already exists", myWindow)
					return
				}
			}
			path := filepath.Join("configs", name+".json")
			configManager.Configs = append(configManager.Configs, ConfigEntry{Name: name, Path: path})
			configManager.CurrentConfigIndex = len(configManager.Configs) - 1
			configManager.CurrentConfig = AppConfig{LogLevel: "Error", Proxies: []ProxyConfig{}, Excludes: []string{}}
			configManager.SaveCurrentConfig()
			refreshConfigSelector()
			configSelector.SetSelected(name)
			logLevelSelect.SetSelected(configManager.CurrentConfig.LogLevel)
			selectedProxyIndex = -1
			proxyList.Refresh()
			updateProxyEditor(-1)
			excludesEntry.SetText("")
			dialog.ShowInformation("Success", "Config '"+name+"' created successfully!", myWindow)
		}, myWindow)
	})

	cloneConfigBtn := widget.NewButton("Clone Config", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("New config name")
		content := container.NewVBox(
			widget.NewLabel("Enter name for cloned config:"),
			widget.NewLabel("Cloning: "+configManager.Configs[configManager.CurrentConfigIndex].Name),
			nameEntry,
		)
		dialog.ShowCustomConfirm("Clone Config", "Clone", "Cancel", content, func(ok bool) {
			if !ok {
				return
			}
			newName := strings.TrimSpace(nameEntry.Text)
			if newName == "" {
				dialog.ShowInformation("Error", "Config name cannot be empty", myWindow)
				return
			}
			for _, c := range configManager.Configs {
				if c.Name == newName {
					dialog.ShowInformation("Error", "Config with this name already exists", myWindow)
					return
				}
			}
			// Read current config
			data, err := os.ReadFile(configManager.Configs[configManager.CurrentConfigIndex].Path)
			if err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			// Save to new config
			newPath := filepath.Join("configs", newName+".json")
			if err := os.WriteFile(newPath, data, 0644); err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			configManager.Configs = append(configManager.Configs, ConfigEntry{Name: newName, Path: newPath})
			refreshConfigSelector()
			configSelector.SetSelected(newName)
			dialog.ShowInformation("Success", "Config cloned successfully!", myWindow)
		}, myWindow)
	})

	renameConfigBtn := widget.NewButton("Rename Config", func() {
		if len(configManager.Configs) == 0 {
			dialog.ShowInformation("Error", "No config to rename", myWindow)
			return
		}
		currentName := configManager.Configs[configManager.CurrentConfigIndex].Name
		if currentName == "default" {
			dialog.ShowInformation("Error", "Cannot rename default config", myWindow)
			return
		}
		nameEntry := widget.NewEntry()
		nameEntry.SetText(currentName)
		content := container.NewVBox(
			widget.NewLabel("Enter new config name:"),
			nameEntry,
		)
		dialog.ShowCustomConfirm("Rename Config", "Rename", "Cancel", content, func(ok bool) {
			if !ok {
				return
			}
			newName := strings.TrimSpace(nameEntry.Text)
			if newName == "" {
				dialog.ShowInformation("Error", "Config name cannot be empty", myWindow)
				return
			}
			if newName == currentName {
				return
			}
			for _, c := range configManager.Configs {
				if c.Name == newName {
					dialog.ShowInformation("Error", "Config with this name already exists", myWindow)
					return
				}
			}
			// Rename file
			oldPath := configManager.Configs[configManager.CurrentConfigIndex].Path
			newPath := filepath.Join("configs", newName+".json")
			if err := os.Rename(oldPath, newPath); err != nil {
				dialog.ShowError(err, myWindow)
				return
			}
			configManager.Configs[configManager.CurrentConfigIndex].Name = newName
			configManager.Configs[configManager.CurrentConfigIndex].Path = newPath
			refreshConfigSelector()
			configSelector.SetSelected(newName)
			dialog.ShowInformation("Success", "Config renamed successfully!", myWindow)
		}, myWindow)
	})

	removeConfigBtn := widget.NewButton("Remove Config", func() {
		if len(configManager.Configs) <= 1 {
			dialog.ShowInformation("Error", "Cannot remove the last config", myWindow)
			return
		}
		dialog.ShowConfirm("Remove Config", "Are you sure?", func(confirmed bool) {
			if !confirmed {
				return
			}
			os.Remove(configManager.Configs[configManager.CurrentConfigIndex].Path)
			configManager.Configs = append(configManager.Configs[:configManager.CurrentConfigIndex], configManager.Configs[configManager.CurrentConfigIndex+1:]...)
			if configManager.CurrentConfigIndex >= len(configManager.Configs) {
				configManager.CurrentConfigIndex = len(configManager.Configs) - 1
			}
			configManager.LoadCurrentConfig()
			refreshConfigSelector()
			configSelector.SetSelected(configManager.Configs[configManager.CurrentConfigIndex].Name)
			logLevelSelect.SetSelected(configManager.CurrentConfig.LogLevel)
			selectedProxyIndex = -1
			proxyList.Refresh()
			updateProxyEditor(-1)
			if len(configManager.CurrentConfig.Excludes) > 0 {
				excludesEntry.SetText(strings.Join(configManager.CurrentConfig.Excludes, "\n"))
			} else {
				excludesEntry.SetText("")
			}
		}, myWindow)
	})
	// Service control buttons
	installServiceBtn := widget.NewButton("Install Service", func() {
		if err := installService(); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to install service: %v", err), myWindow)
		} else {
			dialog.ShowInformation("Success", "Service installed and started successfully!", myWindow)
		}
	})

	uninstallServiceBtn := widget.NewButton("Uninstall Service", func() {
		dialog.ShowConfirm("Uninstall Service", "This will stop and remove ProxiFyre from Windows services. Continue?", func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := uninstallService(); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to uninstall service: %v", err), myWindow)
			} else {
				dialog.ShowInformation("Success", "Service uninstalled successfully!", myWindow)
			}
		}, myWindow)
	})

	proxifyrePathEntry := widget.NewEntry()
	proxifyrePathEntry.SetText(configManager.ProxifyrePath)
	applyPathBtn := widget.NewButton("Apply", func() {
		if err := configManager.SetProxifyrePath(proxifyrePathEntry.Text); err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		configManager.SaveCurrentConfig()
		dialog.ShowInformation("Success", "Path applied and saved.", myWindow)
	})
	detectBtn := widget.NewButton("Detect Path", func() {
		configManager.DetectProxifyrePath()
		proxifyrePathEntry.SetText(configManager.ProxifyrePath)
	})

	topBar := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Log Level:"), logLevelSelect,
			layout.NewSpacer(),
			widget.NewLabel("Config:"), configSelector,
			addConfigBtn, cloneConfigBtn, renameConfigBtn, removeConfigBtn,
		),
		widget.NewForm(
			widget.NewFormItem("ProxiFyre Path", container.NewBorder(nil, nil, nil, container.NewHBox(detectBtn, applyPathBtn), proxifyrePathEntry)),
		),
		widget.NewSeparator(),
	)

	bottomBar := container.NewHBox(
		layout.NewSpacer(),
		loadBtn,
		saveBtn,
		saveAndRestartBtn,
		layout.NewSpacer(),
		installServiceBtn,
		uninstallServiceBtn,
		layout.NewSpacer(),
		exitBtn,
	)

	content := container.NewBorder(
		topBar,
		bottomBar,
		nil,
		nil,
		split,
	)

	myWindow.SetContent(content)
	myWindow.Show()
	myWindow.ShowAndRun()
}

// resolveProxiFyrePath returns a verified path to ProxiFyre.exe
func resolveProxiFyrePath() (string, error) {
	if configManager.ProxifyrePath != "" {
		if _, err := os.Stat(configManager.ProxifyrePath); err == nil {
			return configManager.ProxifyrePath, nil
		}
	}

	// Пробуем авто-поиск
	configManager.DetectProxifyrePath()
	if configManager.ProxifyrePath == "" {
		return "", fmt.Errorf("ProxiFyre.exe not found. Please specify the path in the UI")
	}
	if _, err := os.Stat(configManager.ProxifyrePath); os.IsNotExist(err) {
		return "", fmt.Errorf("ProxiFyre.exe not found at %s", configManager.ProxifyrePath)
	}
	return configManager.ProxifyrePath, nil
}

func (cm *ConfigManager) DeployConfigToProxiFyre() error {
	// Get ProxiFyre directory
	if cm.ProxifyrePath == "" {
		cm.DetectProxifyrePath()
		if cm.ProxifyrePath == "" {
			return fmt.Errorf("ProxiFyre path is not set. Please detect or specify it first.")
		}
	}
	proxifyreDir := filepath.Dir(cm.ProxifyrePath)
	appConfigPath := filepath.Join(proxifyreDir, "app-config.json")
	backupPath := filepath.Join(proxifyreDir, "app-config.json.backup")

	// Backup existing config
	if _, err := os.Stat(appConfigPath); err == nil {
		if err := os.Rename(appConfigPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing config: %v", err)
		}
	}

	// Copy current config to ProxiFyre directory
	data, err := json.MarshalIndent(cm.CurrentConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(appConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to ProxiFyre directory: %v", err)
	}

	return nil
}

func stopService() error {
	exe, err := resolveProxiFyrePath()
	if err != nil {
		return err
	}
	log.Printf("[Service] Stopping via: %s stop", exe)
	cmd := exec.Command(exe, "stop")
	output, err := cmd.CombinedOutput()
	log.Printf("[Service] Stop output: %s", strings.TrimSpace(string(output)))

	// Topshelf возвращает ошибку, если служба не запущена. Логируем, но не прерываем.
	if err != nil {
		log.Printf("[Service] Stop warning (ignored): %v", err)
	}
	return nil
}

func startService() error {
	exe, err := resolveProxiFyrePath()
	if err != nil {
		return err
	}
	log.Printf("[Service] Starting via: %s start", exe)
	cmd := exec.Command(exe, "start")
	out, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to start service: %v\nOutput: %s", err, strings.TrimSpace(string(out)))
	}
	log.Printf("[Service] Start output: %s", strings.TrimSpace(string(out)))
	return nil
}

// runElevated запускает команду с гарантированным запросом прав администратора (UAC)
func runElevated(exe string, args ...string) error {
	if runtime.GOOS != "windows" {
		return exec.Command(exe, args...).Run()
	}
	// PowerShell Start-Process с -Verb runas принудительно вызывает UAC
	// -Wait ждёт завершения процесса, -WindowStyle Hidden скрывает окно консоли
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("Start-Process '%s' -ArgumentList '%s' -Verb runas -Wait -WindowStyle Hidden",
			strings.ReplaceAll(exe, "'", "''"),
			strings.Join(args, " ")))

	// Скрываем само окно PowerShell
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func installService() error {
	exe, err := resolveProxiFyrePath()
	if err != nil {
		return err
	}

	// Запускаем установку с повышением прав
	if err := runElevated(exe, "install"); err != nil {
		// Ошибка возникает, если пользователь нажал "Нет" в UAC или процесс завершился с ошибкой
		return fmt.Errorf("Установка прервана или требует подтверждения прав администратора.\nПожалуйста, разрешите доступ в окне UAC.")
	}

	// Небольшая задержка, чтобы Windows SCM успел зарегистрировать службу
	time.Sleep(500 * time.Millisecond)
	return startService()
}

func uninstallService() error {
	exe, err := resolveProxiFyrePath()
	if err != nil {
		return err
	}

	// Topshelf автоматически останавливает службу при удалении. Отдельный stop не нужен.
	if err := runElevated(exe, "uninstall"); err != nil {
		return fmt.Errorf("Удаление прервано или требует подтверждения прав администратора.\nПожалуйста, разрешите доступ в окне UAC.")
	}
	return nil
}

// restartService перезапускает установленную службу
func restartService() error {
	if err := stopService(); err != nil {
		return err // stopService сам по себе не возвращает ошибок, но оставим для согласованности
	}
	return startService()
}
