// Hellofs implements a simple "hello world" file system.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
	"golang.org/x/net/context"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("helloworld"),
		fuse.Subtype("hellofs"),
		fuse.LocalVolume(),
		fuse.VolumeName("Hello world!"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fs.Serve(c, FS{})
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// FS implements the hello world file system.
type FS struct{}

func (FS) Root() (fs.Node, error) {
	return Dir{}, nil
}

// Dir implements both Node and Handle for the root directory.
type Dir struct{}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}
/*
func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	fmt.Println("Lookup", name)
	if name == "hello" {
		node:=File{}
		return node, nil
	}
	return nil, fuse.ENOENT
}
*/
func (Dir) Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (fs.Node, error) {
	fmt.Println("Lookup", req.Name)
	resp.EntryValid = 0
	if req.Name == "hello" {
		node:=File{}
		return node, nil
	}
	return nil, fuse.ENOENT
}

var dirDirs = []fuse.Dirent{
	{Inode: 2, Name: "hello", Type: fuse.DT_File},
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll")
	return dirDirs, nil
}

// File implements both Node and Handle for the hello file.
type File struct{}

const greeting = "hello, world\n"

func (File) Attr(ctx context.Context, a *fuse.Attr) error {
	for i:=0;i<5;i++{
		_,f,l,ok:=runtime.Caller(i)
		if !ok{
			break
		}
		fmt.Println(f,l)
	}
	fmt.Println("Get attr")
	a.Inode = 2
	a.Mode = 0444
	a.Size = uint64(len(greeting))
	a.Valid = 0
	return nil
}

func (File) ReadAll(ctx context.Context) ([]byte, error) {
	return []byte(greeting), nil
}
