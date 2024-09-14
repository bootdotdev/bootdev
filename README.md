<p align="center">
  <img src="https://github.com/bootdotdev/bootdev/assets/4583705/7a1184f1-bb43-45fa-a363-f18f8309056f" />
</p>

# Bootdev CLI

The official command line tool for [Boot.dev](https://www.boot.dev). It allows you to submit lessons and do other such nonsense.

⭐ Hit the repo with a star if you're enjoying Boot.dev ⭐

## Installation

### 1. Install Go 1.22 or later

The Boot.dev CLI requires a Golang installation, and only works on Linux and Mac. If you're on Windows, you'll need to use WSL. Make sure you install go in your Linux/WSL terminal, not your Windows terminal/UI. There are two options:

**Option 1**: [The webi installer](https://webinstall.dev/golang/) is the simplest way for most people. Just run this in your terminal:

```bash
curl -sS https://webi.sh/golang | sh
```

_Read the output of the command and follow any instructions._

**Option 2**: Use the [official installation instructions](https://go.dev/doc/install).

Run `go version` on your command line to make sure the installation worked. If it did, _move on to step 2_.

**Optional troubleshooting:**

- If you already had Go installed with webi, you should be able to run the same webi command to update it.
- If you already had a version of Go installed a different way, you can use `which go` to find out where it is installed, and remove the old version manually.
- If you're getting a "command not found" error after installation, it's most likely because the directory containing the `go` program isn't in your [`PATH`](https://opensource.com/article/17/6/set-path-linux). You need to add the directory to your `PATH` by modifying your shell's configuration file. First, you need to know _where_ the `go` command was installed. It might be in:

- `~/.local/opt/go/bin` (webi)
- `/usr/local/go/bin` (official installation)
- Somewhere else?

You can ensure it exists by attempting to run `go` using its full filepath. For example, if you think it's in `~/.local/opt/go/bin`, you can run `~/.local/opt/go/bin/go version`. If that works, then you just need to add `~/.local/opt/go/bin` to your `PATH` and reload your shell:

```bash
# For Linux/WSL
echo 'export PATH=$PATH:$HOME/.local/opt/go/bin' >> ~/.bashrc
# next, reload your shell configuration
source ~/.bashrc
```

```bash
# For Mac OS
echo 'export PATH=$PATH:$HOME/.local/opt/go/bin' >> ~/.zshrc
# next, reload your shell configuration
source ~/.zshrc
```

### 2. Install the Boot.dev CLI

This command will download, build, and install the `bootdev` command into your Go toolchain's `bin` directory. Go ahead and run it:

```bash
go install github.com/bootdotdev/bootdev@latest
```

Run `bootdev --version` on your command line to make sure the installation worked. If it did, _move on to step 3_.

_If it worked, move on to step 3._

**Optional troubleshooting:**

If you're getting a "command not found" error for `bootdev help`, it's most likely because the directory containing the `bootdev` program isn't in your [`PATH`](https://opensource.com/article/17/6/set-path-linux). You need to add the directory to your `PATH` by modifying your shell's configuration file. You probably need to add `$HOME/go/bin` (the default `GOBIN` directory where `go` installs programs) to your `PATH`:

```bash
# For Linux/WSL
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
# next, reload your shell configuration
source ~/.bashrc
```

```bash
# For Mac OS
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
# next, reload your shell configuration
source ~/.zshrc
```

### 3. Login to the CLI

Run `bootdev login` to authenticate with your Boot.dev account. After authenticating, you're ready to go!
