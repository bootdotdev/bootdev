<p align="center">
  <img  src="https://www.boot.dev/_nuxt/bootdev-logo-full-small.T5Eqr5qH.png">
</p>

# Bootdev CLI

The official command line tool for [Boot.dev](https://www.boot.dev). It allows you to submit lessons and do other such nonsense.

⭐ Hit the repo with a star if you're enjoying Boot.dev ⭐

## Installation

Make sure you have [Go 1.22 or later installed](https://go.dev/doc/install) on your machine. Additionally, make sure that Go's bin directory is in your PATH. (Details on adding the bin directory to your PATH can be found below)

```bash
go install github.com/bootdotdev/bootdev@latest
```

Make sure that it works by running:

```bash
bootdev help
```

Then, while logged in on the Boot.dev website, authenticate your CLI with:

```bash
bootdev login
```

## Usage

* `bootdev login` - Login to [Boot.dev](https://www.boot.dev). You'll need to login to Boot.dev in your browser and copy/paste a token.
* `bootdev logout` - Logout of Boot.dev (clears your authentication token).
* `bootdev run <id>` - Run a lesson locally to debug your solution.
* `bootdev submit <id>` - Submit a lesson to Boot.dev.

After a `submit` command, results are sent to Boot.dev's servers, and then websocketed to your browser instantly, so be sure to check there after submission.

## How to add Go's bin directory to your PATH

When you run [go install](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies), Go installs the binary into `$HOME/go/bin` by default. Add the bin directory to your PATH by modifying your shell's configuration file. For example, if you're using bash on Ubuntu (e.g. WSL), you can run the following commands to add a line to your `~/.bashrc` file:

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


Now you should be able to run the `bootdev` command (or anything else installed with `go install`) from your terminal.
