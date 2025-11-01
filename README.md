# 🧹 Janitor

A powerful and intuitive CLI tool for cleaning up your development environment. Janitor helps you identify and remove junk directories like `node_modules`, `target`, `build`, and clean global package caches with ease.

## ✨ Features

- **🎨 Interactive TUI Interface**: User-friendly text-based interface for managing cleanup tasks
- **🔄 Real-time Scanning Progress**: Live statistics showing directories scanned and junk found
- **🗑️ "Trash First" Safety**: All deletions moved to recycle bin by default (unless `--force` is used)
- **🚀 Non-Interactive CLI Mode**: Perfect for automation, CI/CD pipelines, or scripts
- **🧰 Global Cache Cleaning**: Clean caches for Docker, npm, pip, Go, Cargo, and more
- **📝 `.janitorignore` Support**: Project-specific ignore rules (like `.gitignore`)
- **⏰ Advanced Age-based Rules**: Filter items by last modification time
- **🎯 Customizable Cleaners**: Define your own custom cleaning commands

## 📦 Installation

### Download Pre-built Binary

Download the latest release from the [Releases](../../releases) page.

### Build from Source

**Requirements:**
- Go 1.20 or higher

```bash
git clone https://github.com/J-ShuJie/janitor.git
cd janitor
go build -o janitor
```

### Install via Go

```bash
go install github.com/J-ShuJie/janitor@latest
```

## 🚀 Quick Start

### Interactive TUI Mode

Simply run without arguments to launch the interactive interface:

```bash
janitor
```

Navigate with arrow keys, select options, and manage your cleanup tasks interactively.

### Scan for Junk

Scan current directory for junk:

```bash
janitor scan
```

Scan specific directory:

```bash
janitor scan /path/to/project
```

Preview what would be deleted (dry run):

```bash
janitor scan --dry-run
```

Actually delete found items (moves to trash):

```bash
janitor scan --delete
```

Permanently delete (⚠️ use with caution):

```bash
janitor scan --delete --force
```

### Clean Global Caches

List available cache cleaners:

```bash
janitor cleancache
```

Clean all detected caches:

```bash
janitor cleancache --all
```

Clean specific cache:

```bash
janitor cleancache docker
```

## ⚙️ Configuration

### Initialize Configuration

```bash
janitor config init
```

This creates a configuration file at:
- **Windows**: `%USERPROFILE%\AppData\Roaming\janitor\config.yml`
- **macOS/Linux**: `~/.config/janitor/config.yml`

### Edit Configuration

```bash
janitor config edit
```

### Configuration Example

```yaml
# Directories to scan for
scan:
  directories:
    - node_modules
    - target
    - build
    - dist
    - venv
    - __pycache__
    - .gradle
    - DerivedData

  # Advanced rules
  rules:
    # Only clean directories older than 30 days (0 = clean all)
    older_than_days: 30

# Global ignore paths (protected zones)
ignore_paths:
  - /System
  - /Library
  - /Applications
  - C:\Windows
  - ~\Backups

# Custom cleaners
custom_cleaners:
  - name: "Clear Temp Files"
    estimate_command: "du -sh /tmp"
    clean_command: "rm -rf /tmp/*"
    requires_sudo: true
```

## 📝 `.janitorignore` File

Create a `.janitorignore` file in any directory to protect specific paths:

```gitignore
# Protect important node_modules
important-project/node_modules

# Protect all build directories
build/

# Use glob patterns
*.log
test-*/
```

**Pattern syntax (gitignore-style):**
- `node_modules` - matches directory anywhere
- `build/` - matches directory named "build"
- `/temp` - matches only at root level
- `*.log` - wildcard patterns
- `!important.log` - negation patterns

## 🎯 Use Cases

### Clean Old Projects

Remove junk from projects you haven't touched in months:

```yaml
# config.yml
scan:
  rules:
    older_than_days: 90  # 3 months
```

```bash
janitor scan ~/projects
```

### CI/CD Pipeline

```bash
# Clean up after build
janitor scan . --delete --dry-run  # Preview first
janitor scan . --delete            # Actually clean
```

### Free Up Disk Space

```bash
# Check what's taking space
janitor scan / --dry-run

# Clean global caches
janitor cleancache --all
```

### Before/After Development

```bash
# Before: Clean old junk
janitor scan --delete

# After: Clean build artifacts
janitor scan . --delete
```

## 🔒 Safety Features

1. **Trash First**: By default, deleted items go to recycle bin/trash
2. **Guardrails**: Protected system paths cannot be deleted
3. **Dry Run Mode**: Preview before deletion
4. **Confirmation Dialogs**: Interactive confirmations in TUI mode
5. **Age-based Filtering**: Only clean old directories

## 🧪 Testing

Run automated tests:

```bash
# Windows
test\auto_test.bat

# macOS/Linux
bash test/auto_test.sh
```

Run unit tests:

```bash
go test ./...
```

## 🏗️ Project Structure

```
janitor/
├── main.go              # Entry point
├── cmd/                 # CLI commands
│   ├── scan.go         # Scan command
│   ├── clean.go        # Cache cleaning
│   └── config.go       # Configuration
├── internal/
│   ├── config/         # Config management
│   ├── core/           # Core logic
│   │   ├── scanner.go  # Directory scanning
│   │   ├── cleaner.go  # Cache cleaning
│   │   ├── safety.go   # Safety checks
│   │   └── ignore/     # .janitorignore parsing
│   ├── logger/         # Logging
│   └── tui/            # Terminal UI
└── test/               # Tests
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling

## 💡 Tips

- Run `janitor scan --dry-run` first to preview what would be deleted
- Use `.janitorignore` files to protect important directories
- Set `older_than_days: 0` in config to clean all projects regardless of age
- Use `--force` flag only when you're absolutely sure
- Check trash/recycle bin after deletion if you need to restore something

## 📊 Example Output

```
Scanning for junk...
Scanned 1,234 directories, found 2.5 GB junk (12 items matching rules)

Found junk items:
- /home/user/old-project/node_modules (856.3 MB) (Last Modified 120 days)
- /home/user/test-app/target (1.2 GB) (Last Modified 90 days)
- /home/user/demo/build (456.7 MB) (Last Modified 60 days)

Use --delete to remove items, --dry-run to see what would be deleted.
```

## ⚠️ Important Notes

- **Always backup important data** before running cleanup operations
- The `--force` flag bypasses the recycle bin and **permanently deletes** files
- Protected system paths (like `/System`, `C:\Windows`) are automatically excluded
- Items not meeting age rules are displayed in grey and cannot be selected in TUI mode

---

**Made with ❤️ for developers who hate disk space bloat**
