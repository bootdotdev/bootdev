# Bootdev CLI

This is a command line tool for [Boot.dev](https://boot.dev) that allows you to submit lessons and do other such nonsense.

## Installation

Make sure you have [Go 1.22 or later installed](https://go.dev/doc/install) on your machine. Additionally, make sure that Go's bin directory is in your PATH.

```bash
go install github.com/bootdotdev/bootdev@latest
```

## Usage

* `bootdev login` - Login to Boot.dev. You'll need to login to Boot.dev in your browser and copy/paste a token.
* `bootdev logout` - Logout of Boot.dev (clears your authentication token).
* `bootdev submit <id>` - Submit a lesson to Boot.dev.
