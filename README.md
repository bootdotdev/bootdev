<p align="center">
  <img
    src="https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/wvnt9yl-800x370.png"
    alt="Boot.dev logo"
    style="width: 500px"
  />
</p>

<h1 align="center">Boot.dev CLI</h1>

<p align="center">
  The official command line tool for <a href="https://www.boot.dev/">Boot.dev</a>.<br />
  Submit lessons and do other such nonsense, straight from your terminal.
</p>

<p align="center">
  ⭐ Hit the repo with a star if you're enjoying Boot.dev ⭐
</p>

---

## Table of Contents

- [Prerequisites](#prerequisites)
  - [Platform Support](#platform-support)
- [Installation](#installation)
  - [Step 1 — Install Go](#step-1--install-go)
  - [Step 2 — Install the Boot.dev CLI](#step-2--install-the-bootdev-cli)
  - [Step 3 — Log In to the CLI](#step-3--log-in-to-the-cli)
- [Configuration](#configuration)
  - [Where the Config Lives](#where-the-config-lives)
  - [Base URL for HTTP Tests](#base-url-for-http-tests)
  - [CLI Colors](#cli-colors)
- [Upgrading](#upgrading)
- [Troubleshooting](#troubleshooting)
  - [Go Installation Problems](#go-installation-problems)
  - [`bootdev` Command Not Found](#bootdev-command-not-found)
  - [Configuration Problems](#configuration-problems)
  - [Upgrade Problems](#upgrade-problems)

---

## Prerequisites

Before installing the CLI, make sure you have:

- An up-to-date **Golang toolchain** installed on your system.
- A **Boot.dev account** to log in with.

### Platform Support

> [!IMPORTANT]
> Check which platform your course actually targets before you start.

| Platform | Notes |
| --- | --- |
| **Linux** | Fully supported. |
| **macOS** | Fully supported. |
| **WSL (Linux-in-Windows)** | The usual choice for Windows users — go into WSL and follow the Linux instructions. |
| **Windows / PowerShell** | The overwhelming majority of courses that use this CLI are designed for Linux/macOS/WSL, but there is now [at least one Windows-native course](https://www.boot.dev/courses/learn-data-visualization-power-bi). Windows/PowerShell instructions are included below. |

---

## Installation

### Step 1 — Install Go

There are two recommended ways to get the Go toolchain. Pick the one that matches your platform.

#### Option 1 — Webi installer (Linux / WSL / macOS)

The [Webi installer](https://webinstall.dev/golang/) is the simplest method for most people. Run this in your terminal:

```sh
curl -sS https://webi.sh/golang | sh
```

> [!NOTE]
> Read the output of the command and follow any instructions it gives you.

#### Option 2 — Official installer (any platform, including Windows / PowerShell)

Follow the [official Golang installation instructions](https://go.dev/doc/install). On Windows, this means downloading and running a `.msi` installer package; the rest should be taken care of automatically.

#### Verify the installation

After installing Golang, **open a new shell session** and confirm everything works:

```sh
go version
```

> [!TIP]
> If that prints a version, move on to [Step 2](#step-2--install-the-bootdev-cli). If not, see [Go Installation Problems](#go-installation-problems).

---

### Step 2 — Install the Boot.dev CLI

This command downloads, builds, and installs the `bootdev` program into your Go toolchain's `bin` directory:

```sh
go install github.com/bootdotdev/bootdev@latest
```

Then confirm the installation worked:

```sh
bootdev --version
```

> [!TIP]
> If that prints a version, move on to [Step 3](#step-3--log-in-to-the-cli). If you get a "command not found" error, see [`bootdev` Command Not Found](#bootdev-command-not-found).

---

### Step 3 — Log In to the CLI

Authenticate with your Boot.dev account:

```sh
bootdev login
```

After authenticating, you're ready to go!

---

## Configuration

The Boot.dev CLI offers a couple of configuration options.

> [!NOTE]
> All commands have `-h` / `--help` flags if you want to see the available options on the command line.

### Where the Config Lives

Configuration is stored in a config file:

| Condition | Config file location |
| --- | --- |
| Default | `~/.bootdev.yaml` |
| If `XDG_CONFIG_HOME` is set | `$XDG_CONFIG_HOME/bootdev/config.yaml` |

### Base URL for HTTP Tests

For lessons with HTTP tests, you can configure the CLI with a base URL that overrides any lesson's default. A common use case is when you want to run your server on a port other than the one specified in the lesson.

- **Set the base URL:**

  ```sh
  bootdev config base_url YOUR_URL
  ```

  > [!WARNING]
  > Make sure you include the protocol scheme (`http://`) in the URL.

- **Get the current base URL** (the default is an empty string):

  ```sh
  bootdev config base_url
  ```

- **Reset the base URL** and revert to using the lessons' defaults:

  ```sh
  bootdev config base_url --reset
  ```

### CLI Colors

The CLI text output is rendered with extra colors:

| Color | Typically used for |
| --- | --- |
| Green | Success messages |
| Red | Error messages |
| Gray | Secondary text |

- **Customize the colors:**

  ```sh
  bootdev config colors --red VALUE --green VALUE --gray VALUE
  ```

  > [!NOTE]
  > You can use an [ANSI color code](https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit) or a hex string as the `VALUE`.

- **Get the current colors:**

  ```sh
  bootdev config colors
  ```

- **Reset the colors** to their default values:

  ```sh
  bootdev config colors --reset
  ```

---

## Upgrading

> [!NOTE]
> If you just installed the CLI, it's already upgraded!

The Boot.dev CLI is regularly updated to enhance and expand its features and integration with the web app. The CLI automatically detects new versions and will require you to upgrade it before submitting or logging in.

- **Recommended:** use the built-in upgrade command:

  ```sh
  bootdev upgrade
  ```

- **Alternative:** use `go install` with the [latest tagged version](https://github.com/bootdotdev/bootdev/tags):

  ```sh
  go install github.com/bootdotdev/bootdev@v1.XX.X
  ```

---

## Troubleshooting

### Go Installation Problems

- **Already installed Go with Webi?** You should be able to run the same Webi command again to update it.
- **Already installed Go a different way?** On Linux/macOS, find out where it lives and (if needed) remove the old version manually:

  ```sh
  which go
  ```

  In PowerShell on Windows, the equivalent is:

  ```powershell
  Get-Command go
  ```

- **Getting a "command not found" error after installation?** It's most likely because the directory containing the `go` program isn't in your [`PATH`](https://opensource.com/article/17/6/set-path-linux). You'll need to add that directory to your `PATH` by modifying your shell's configuration file.

  First, figure out *where* the `go` command was installed. It might be in:

  | Installation method | Likely location |
  | --- | --- |
  | Webi | `~/.local/opt/go/bin` |
  | Official installation | `/usr/local/go/bin` |
  | Something else | Somewhere else? |

  You can confirm the program exists by running `go` using its full filepath. For example, if you think it's in `~/.local/opt/go/bin`:

  ```sh
  ~/.local/opt/go/bin/go version
  ```

  If that works, you just need to add `~/.local/opt/go/bin` to your `PATH` and reload your shell:

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

  ```sh
  # For fish
  fish_add_path $HOME/.local/opt/go/bin
  ```

### `bootdev` Command Not Found

If you're getting a "command not found" error for `bootdev`, it's likely because the directory containing the `bootdev` program isn't in your [`PATH`](https://opensource.com/article/17/6/set-path-linux). You need to add the directory to `PATH` by modifying your shell's configuration file.

In most cases, this means adding `$HOME/go/bin` — the default `GOBIN` directory where `go` installs programs:

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

```sh
# For fish
fish_add_path $HOME/go/bin
```

### Configuration Problems

If you want to undo changes to your configuration, simply remove it to reset it completely. The CLI will automatically create a fresh config file in the original location. Then log in again.

### Upgrade Problems

#### Bypass the proxy

If you keep getting the same upgrade message, you may be pulling from an old cache:

```sh
GOPROXY=direct go install github.com/bootdotdev/bootdev@v1.XX.X
```

#### Reinstall

If that doesn't work, try a fresh install:

1. **Locate the binary file:**

   ```sh
   which bootdev
   ```

2. **Carefully remove the binary file** after confirming the path is correct:

   ```sh
   rm "$(which bootdev)"
   ```

   > [!WARNING]
   > Double-check the path before removing anything.

3. **Confirm the binary is gone.** It could be installed in multiple locations:

   ```sh
   which bootdev
   ```

4. **Clean install.** Repeat the steps you used to install the CLI — see [Installation](#installation). Then log in again.
