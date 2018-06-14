# uafs
UA Cloud filesystem in user space.

# Installation

Before download install [fuse](https://docs.oracle.com/cd/E76382_01/bigData.Doc/install_onPrem/src/tins_prereq_index_fuse.html) and [Golang](https://golang.org/dl/) dependences.

```bash
curl -s http://d.librelabua.org/uafs-install.sh | bash
```

# Usage

```bash
$ # uafs [your user with or without '@alu.ua.es'] [mountpoint]
$ uafs pako2 /tmp/uacloud  # mounting
$ fusermount -u /tmp/uacloud # unmounting
```

This filesystem does not support `mv` operations (only `cp`)
