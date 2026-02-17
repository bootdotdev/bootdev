<p align="center">
  <img src="https://github.com/bootdotdev/bootdev/assets/4583705/7a1184f1-bb43-45fa-a363-f18f8309056f" />
</p>

# Boot.dev CLI

This is the official command line tool for [Boot.dev](https://www.boot.dev/). It allows you to submit lessons and do other such nonsense.

⭐ Hit the repo with a star if you're enjoying Boot.dev ⭐

## Installation

### 1. Install Go

To use the Boot.dev CLI, you need an up-to-date Golang toolchain installed on your system.

Please note, the overwhelming majority of our courses that use this CLI are designed to be completed on Linux or macOS – or on Linux-in-Windows via WSL. If you're on Windows, _usually_ what you'll want is to go into WSL and follow Linux installation instructions. However, we now have [at least one course](https://www.boot.dev/courses/learn-data-visualization-power-bi) that is Windows-native. So there are also Windows/PowerShell installation instructions below. Just be aware of which platform you're actually using!

There are two main installation methods that we recommend:

**Option 1 (Linux/WSL/macOS):** The [Webi installer](https://webinstall.dev/golang/) is the simplest way for most people. Just run this in your terminal:

```sh
curl -sS https://webi.sh/golang | sh
```

_Read the output of the command and follow any instructions._

**Option 2 (any platform, including Windows/PowerShell):** Use the [official Golang installation instructions](https://go.dev/doc/install). On Windows, this means downloading and running a `.msi` installer package; the rest should be taken care of automatically.

After installing Golang, _open a new shell session_ and run `go version` to make sure everything works. If it does, _move on to step 2_.

**Optional troubleshooting:**

- If you already had Go installed with Webi, you should be able to run the same Webi command to update it.
- If you already had a version of Go installed a different way, on Linux/macOS you can run `which go` to find out where it's installed, and (if needed) remove the old version manually. (In PowerShell on Windows, the equivalent is `Get-Command go`.)
- If you're getting a "command not found" error after installation, it's most likely because the directory containing the `go` program isn't in your [`PATH`](https://opensource.com/article/17/6/set-path-linux). You need to add the directory to your `PATH` by modifying your shell's configuration file. First, you need to know _where_ the `go` command was installed. It might be in:
  - `~/.local/opt/go/bin` (Webi)
  - `/usr/local/go/bin` (official installation)
  - Somewhere else?

  You can ensure that the program exists by attempting to run `go` using its full filepath. For example, if you think it's in `~/.local/opt/go/bin`, you can run `~/.local/opt/go/bin/go version`. If that works, then you just need to add `~/.local/opt/go/bin` to your `PATH` and reload your shell:

  ```sh
  # For Linux/WSL
  echo 'export PATH=$PATH:$HOME/.local/opt/go/bin' >> ~/.bashrc
  # Next, reload your shell configuration
  source ~/.bashrc
  ```

  ```sh
  # For macOS
  echo 'export PATH=$PATH:$HOME/.local/opt/go/bin' >> ~/.zshrc
  # Next, reload your shell configuration
  source ~/.zshrc
  ```

### 2. Install the Boot.dev CLI

The following command will download, build, and install the `bootdev` command into your Go toolchain's `bin` directory. Go ahead and run it:

```sh
go install github.com/bootdotdev/bootdev@latest
```

Run `bootdev --version` on your command line to make sure the installation worked. If it did, _move on to step 3_.

**Optional troubleshooting:**

If you're getting a "command not found" error for `bootdev help`, it's most likely because the directory containing the `bootdev` program isn't in your [`PATH`](https://opensource.com/article/17/6/set-path-linux). You need to add the directory to your `PATH` by modifying your shell's configuration file. You probably need to add `$HOME/go/bin` (the default `GOBIN` directory where `go` installs programs) to your `PATH`:

```sh
# For Linux/WSL
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
# Next, reload your shell configuration
source ~/.bashrc
```

```sh
# For macOS
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
# Next, reload your shell configuration
source ~/.zshrc
```

### 3. Login to the CLI

Run `bootdev login` to authenticate with your Boot.dev account. After authenticating, you're ready to go!

## Configuration

The Boot.dev CLI offers a couple of configuration options that are stored in a config file (default is `~/.bootdev.yaml`, or `$XDG_CONFIG_HOME/bootdev/config.yaml` if `XDG_CONFIG_HOME` is set).

All commands have `-h`/`--help` flags if you want to see available options on the command line.

### Base URL for HTTP tests

For lessons with HTTP tests, you can configure the CLI with a base URL that overrides any lesson's default. A common use case for that is when you want to run your server on a port other than the one specified in the lesson.

- To set the base URL, run:

  ```sh
  bootdev config base_url <url>
  ```

  _Make sure you include the protocol scheme (`http://`) in the URL._

- To get the current base URL (the default is an empty string), run:

  ```sh
  bootdev config base_url
  ```

- To reset the base URL and revert to using the lessons' defaults, run:

  ```sh
  bootdev config base_url --reset
  ```

### CLI colors

The CLI text output is rendered with extra colors: green (e.g., success messages), red (e.g., error messages), and gray (e.g., secondary text).

- To customize these colors, run:

  ```sh
  bootdev config colors --red <value> --green <value> --gray <value>
  ```

  _You can use an [ANSI color code](https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit) or a hex string as the `<value>`._

- To get the current colors, run:

  ```sh
  bootdev config colors
  ```

- To reset the colors to their default values, run:

  ```sh
  bootdev config colors --reset
  ```
