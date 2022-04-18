# EXPERIMENTAL go source reoreding

This tool is EXPERIMENTAL! We strongly recommend to backup (or use git to commit your changes) before to try it.

This tool will "reorder" your sources to do this:

- alphabetic reorder you methods and constructors (constructors will be also placed above methods)
- place methods and constructors above the `type` definition
- rewrite (or output) the result

Usage:

```bash

Usage of ./goreorder:
  -dir string
    	directory to scan (default: current directory)
  -file string
    	file to process (deactivate -dir flag)
  -format string
    	the executable to use to format the output (default "gofmt")
  -write
    	write the output to the file, if not set it will print to stdout

```

By default, the tool will scan everything in the current directory and output result to standard output (no write).

# Install

Get release or use `go install github.com/metal3d/goreorder@latest`

If you want to install from source:

```
go install -v github.com/metal3d/goreorder/cmd/
```

You can also get this repository and type:

```
make install
```

