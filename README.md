# uafs
UA Cloud filesystem in user space.

# Installation

Linux:

```bash
curl http://d.librelabua.org/uafs-install.sh | bash
```

Go get:

```bash
go get -u -v -x github.com/LibreLABUA/uafs
```

# Dependences

- Requires [fuse](https://docs.oracle.com/cd/E76382_01/bigData.Doc/install_onPrem/src/tins_prereq_index_fuse.html) and [Golang](https://golang.org/dl/)

# Usage

```bash
$ # uafs [your user with or without '@alu.ua.es'] [mountpoint]
$ uafs pako2 /tmp/uacloud  # mounting
$ fusermount -u /tmp/uacloud # unmounting
```

This filesystem does not support `mv` operations (only `cp`)
