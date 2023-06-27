#!/bin/sh

# config
GIT_REPOSITORY=https://github.com/metal3d
GIT_REPONAME=goreorder
GIT_ITEM_PREFIX=goreorder-
BIN_NAME=goreorder

# get kernel name in lowercase
KERNEL_NAME=`uname -s | tr '[:upper:]' '[:lower:]'`

# get the architecture name in lowercase
ARCH=`uname -m | sed -e 's/i.86/i386/' -e 's/x86_64/amd64/'`

# find this in release (e.g. foobar-lunux-amd64)
TARGET=${GIT_ITEM_PREFIX}${KERNEL_NAME}-${ARCH}

# by default, use /usr/local/bin
BIN_PATH="/usr/local/bin"

# if the user is not using sudo or root
if [ "$(id -u)" != "0" ]; then
    # Get HOME bin path that can be $HOME/.local/bin or $HOME/bin
    if [ -d "$HOME/.local/bin" ]; then
        BIN_PATH="$HOME/.local/bin"
    else
        BIN_PATH="$HOME/bin"
    fi

    # check if $BIN_PATH is in the PATH
    if ! echo $PATH | grep -q "$BIN_PATH"; then
        echo "PATH is not set correctly. Please add $BIN_PATH to PATH"
        exit 1
    fi

fi

# download the binary from latest release in github.com/metal3d/goreorder
curl -L --progress-bar ${GIT_REPOSITORY}/${GIT_REPONAME}/releases/latest/download/${TARGET} -o $BIN_PATH/$BIN_NAME && \
    chmod +x $BIN_PATH/$BIN_NAME && \
    echo "goreorder has been installed to $BIN_PATH/$BIN_NAME"
