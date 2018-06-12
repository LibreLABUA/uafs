# uafs
UA Cloud filesystem in user space.

# Installation

```bash
go get -u -v -x github.com/LibreLABUA/uafs
```

# Dependences

- Requires `fuse`

# Usage

```bash
$ # uafs [your user with or without '@alu.ua.es'] [mountpoint]
$ uafs pako2 /tmp/uacloud  # mounting
$ fusermount -u /tmp/uacloud # unmounting
```

This filesystem does not support `mv` operations (only `cp`)
