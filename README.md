# EXPERIMENTAL go source reoreding

This tool is EXPERIMENTAL! We strongly recommend to backup (or use git to commit your changes) before to try it.

This tool will "reorder" your sources to do this:

- alphabetic reorder you methods and constructors (constructors will be also placed above methods)
- place methods and constructors below the `type` definition
- output the result or write or even generate a patch file

# Install

Get release or use `go install github.com/metal3d/goreorder@latest` and download corresponding binary inside your `$PATH`.

You can use this script to install `goreorder` as user or with `sudo`, the script will detect if you are simple user
and will try to install in `$HOME/.local/bin` then `$HOME/bin` it the first one doesn't exists.

```bash
curl -sSL https://raw.githubusercontent.com/metal3d/goreorder/main/repo-tools/install.sh | bash -s
```

You can also get this repository and build it with the `Makefile`:

```bash
git clone git@github.com:metal3d/goreorder.git
cd goreorder
make install
```

## Basic Usage

```
goreorder reorders the structs (optional), methods and constructors in a Go
source file. By default, it will print the result to stdout. To allow goreorder
to write to the file, use the -write flag.

Usage:
  goreorder [flags] [file.go|directory|stdin]
  goreorder [command]

Examples:
$ goreorder reorder --write --reorder-structs --format gofmt file.go
$ goreorder reorder --diff ./mypackage
$ cat file.go | goreorder reorder

Available Commands:
  completion  Generates completion scripts
  help        Help about any command
  reorder     Reorder stucts, methods and constructors in a Go source file.

Flags:
  -h, --help      Show help
  -V, --version   Show version

Use "goreorder [command] --help" for more information about a command.
```

# Avoid destruction with `--diff`

If your system provides `diff` and `patch` command, it is safier to use the `--diff` option to geneate
a `patch` file. This file can then be used to apply changes, and to revert your changes if it fails.

Example:
```bash
goreorder reorder --diff ./ > reorder.patch

# try to apply
patch -p1 --dry-run < ./reorder.patch
# really apply
patch -p1  < ./reorder.patch

# revert the changes
patch -p1 -R < ./reorder.patch
```

# Contribute

Please fill an issue to create a bug report.

If you want to participate, please fork the repository and propose a pull request **on the "develop" branch**.
