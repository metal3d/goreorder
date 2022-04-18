# EXPERIMENTAL go source reoreding

This tool is EXPERIMENTAL! We strongly recommend to backup (or use git to commit your changes) before to try it.

This tool will "reorder" your sources to do this:

- alphabetic reorder you methods and constructors (constructors will be also placed above methods)
- place methods and constructors above the `type` definition
- rewrite (or output) the result

Usage:

```
Usage of goreorder:
  -dir string
    	directory to scan (default ".")
  -file string
    	file to process, deactivates -dir if set
  -format string
    	the executable to use to format the output (default "gofmt")
  -output string
    	output file (default to the original file, only works with -file)
  -reorder-structs
    	reorder structs by name (default: false)
  -verbose
    	get some informations while processing
  -version
    	show version (master)
  -write
    	write the output to the file, if not set it will print to stdout (default: false)
```

By default, the tool will scan everything in the current directory and output result to standard output (no write).

# Install

Get release or use `go install github.com/metal3d/goreorder@latest` and download corresponding binary inside your `$PATH`. You can use this script to install `goreorder` as user or with `sudo`, the script will detect if you are simple user and will try to install in `$HOME/.local/bin` then `$HOME/bin` it the first one doesn't exists.

```bash
curl -sSL https://raw.githubusercontent.com/metal3d/goreorder/main/repo-tools/install.sh | bash
```

If you want to install from source:

```bash
go install -v github.com/metal3d/goreorder/cmd/...
```

You can also get this repository and type:

```bash
git clone git@github.com:metal3d/goreorder.git
cd goreorder
make install
```

# Contribute

Please fill an issue to create a bug report.

If you want to participate, please fork the repository and propose a pull request.
