# Bootdev CLI

This is a command line tool for [Boot.dev](https://www.boot.dev) that allows you to submit lessons and do other such nonsense.

## Installation

Make sure you have [Go 1.22 or later installed](https://go.dev/doc/install) on your machine. Additionally, make sure that Go's bin directory is in your PATH. (Details on adding the bin directory to your PATH can be found below)

```bash
go install github.com/bootdotdev/bootdev@latest
```

Make sure that it works by running:

```bash
bootdev help
```

## Usage

* `bootdev login` - Login to [Boot.dev](https://www.boot.dev). You'll need to login to Boot.dev in your browser and copy/paste a token.
* `bootdev logout` - Logout of Boot.dev (clears your authentication token).
* `bootdev submit <id>` - Submit a lesson to Boot.dev.

## How to add Go's bin directory to your PATH

When you run [go install](https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies), Go installs the binary into `$HOME/go/bin` by default. Add the bin directory to your PATH by modifying your shell's configuration file. For example, if you're using bash on Ubuntu, you can run the following command to add a line to your `~/.bashrc` file:

```bash
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
```

Next, reload your shell configuration:

```bash
source ~/.bashrc
```

Now you should be able to run the `bootdev` command (or anything else installed with `go install`) from your terminal.
