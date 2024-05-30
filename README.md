<p align="center">
  <img src="https://github.com/bootdotdev/bootdev/assets/4583705/7a1184f1-bb43-45fa-a363-f18f8309056f" />
</p>

# Bootdev CLI

The official command line tool for [Boot.dev](https://www.boot.dev). It allows you to submit lessons and do other such nonsense.

⭐ Hit the repo with a star if you're enjoying Boot.dev ⭐

## Installation

### 1. You need Go 1.22 installed

The Boot.dev CLI only works on Linux and Mac. If you're on Windows, you'll need to use WSL. Make sure you install go in your Linux/WSL terminal, not your Windows terminal/UI. We recommend using the [webi instructions here](https://webinstall.dev/golang/) for a quick and easy Go installation on the command line. It's as easy as running this in your terminal:

```bash
curl -sS https://webi.sh/golang | sh
```

Alternatively, you can use the [official installation instructions](https://go.dev/doc/install).

Run `go version` on your command line to make sure the installation worked.

### 2. Install the Boot.dev CLI

This command will download, build, and install the `bootdev` command into your Go toolchain's `bin` directory. Go ahread and run it:

```bash
go install github.com/bootdotdev/bootdev@latest
```

Make sure that it works by running:

```bash
bootdev help
```

### 3. Add to PATH (if you're having issues)

If you're getting a "command not found" error, it's most likely because Go's bin directory (where your `bootdev` command is) isn't in your PATH. You can add the bin directory to your PATH by modifying your shell's configuration file. For example, if you're using bash on Ubuntu (e.g. WSL), you can run the following commands to add a line to your `~/.bashrc` file:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc

# next, reload your shell configuration
source ~/.bashrc
```

Or if you're on Mac OS using zsh:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc

# next, reload your shell configuration
source ~/.zshrc
```

## Usage

The first time you use the tool, run `bootdev login` to authenticate with your Boot.dev account. Here are the other commands:

* `bootdev login` - Login to [Boot.dev](https://www.boot.dev). You'll need to login to Boot.dev in your browser and copy/paste a token.
* `bootdev logout` - Logout of Boot.dev (clears your authentication token).
* `bootdev run <id>` - Run a lesson locally to debug your solution.
* `bootdev submit <id>` - Submit a lesson to Boot.dev.

After a `submit` command, results are sent to Boot.dev's servers, and then websocketed to your browser instantly, so be sure to check there after submission.

