package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"unsafe"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/erikdubbelboer/fasthttp"
	"github.com/howeyc/gopass"
	"github.com/marcsantiago/gocron"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/afero"
	"github.com/themester/fcookiejar"
	"github.com/valyala/fastrand"
)

// max cache files
var (
	maxFiles    = flag.Int("n", 5, "Max files")
	cacheUpdate = flag.Uint64("u", 2, "Cache update time")
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("%s <username>@alu.ua.es <mount point>\n", os.Args[0])
		os.Exit(0)
	}
	// TODO: key daemon process
	pass := os.Getenv("psswrd")
	if pass == "" {
		fmt.Printf("Password: ")
		p, err := gopass.GetPasswd()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pass = b2s(p)
	}
	if !strings.Contains(os.Args[1], "@alu") {
		os.Args[1] += "@alu.ua.es"
	}

	// logging in
	client, cookies, err := login(
		os.Args[1], pass,
	)
	if err != nil {
		// bad password or client error
		fmt.Println(err)
		os.Exit(1)
	}

	// invoking daemon
	ctx := daemon.Context{
		Env:         append(os.Environ(), "psswrd="+pass),
		LogFileName: path.Join(os.TempDir(), "uafs.log"),
	}
	dmn, err := ctx.Reborn()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if dmn != nil { // parent
		return
	}
	defer dmn.Release()

	// stat of mount dir
	created := false
	if _, err := os.Stat(os.Args[2]); err != nil {
		created = true
		os.Mkdir(os.Args[2], 0755)
	}

	// creating virtual filesystem
	root := &FS{
		Cookies: cookies,
		Client:  client,
		Name:    os.Args[1],
		Pass:    pass,
		Fs:      afero.NewMemMapFs(),
		items:   make([]*uaitem, 0),
	}
	root.fetch()

	// mounting fuse system
	fconn, err := fuse.Mount(
		os.Args[2],
		fuse.FSName("uafs"),
		fuse.Subtype("uafs"),
		fuse.VolumeName("ua-volume"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer fconn.Close()

	// see checkFiles function
	// checkFiles function will be done every 5 minutes
	gocron.Every(*cacheUpdate).Minutes().Do(checkFiles, root)
	// serve filesystem connections
	err = fs.Serve(fconn, root)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		os.Remove(os.Args[2])
	}
}

// checkFiles checks cache files
// if file has not been modified in 20 minutes it will be deleted.
func checkFiles(fs *FS) {
	defer fs.fetch()
	// using mutexes
	fs.Lock()
	// copying cache files
	files := fs.downloadFiles
	// resetting files
	fs.downloadFiles = fs.downloadFiles[:0]
	fs.Unlock()
	for i := 0; i < len(files); i++ {
		file := files[i]
		st, err := fs.Fs.Stat(file)
		if err != nil {
			continue
		}
		// getting last modification time
		if time.Since(st.ModTime()) > time.Minute*20 {
			// removing
			fs.Fs.Remove(file)
			file, err := fs.Fs.Create(file)
			if err == nil {
				file.Close()
			}
			files = append(files[:i], files[i+1:]...)
			i--
		}
	}
	// unlocking and adding to cache slice
	fs.Lock()
	fs.downloadFiles = append(fs.downloadFiles, files...)
	fs.Unlock()
}

// filesystem fuse structure
var _ fs.FS = (*FS)(nil)

type FS struct {
	sync.RWMutex
	// cache slice
	downloadFiles []string
	// UACloud cookies
	Cookies *cookiejar.CookieJar
	// UACloud client
	Client *fasthttp.Client
	Name   string
	Pass   string
	// Virtual in-memory filesystem
	Fs afero.Fs
	// downloaded items
	items []*uaitem
}

func (root *FS) fetch() {
	// getting UACloud folders
	root.getFolders()
	for i := range root.items {
		root.getItems(root.items[i])
	}
	root.Fs.RemoveAll("/*")
	root.Fs.Mkdir("/", 0777)
	root.fill("/", root.items)
}

// find item by name (path)
func lookup(items []*uaitem, name string) *uaitem {
	for _, item := range items {
		if path.Join(item.path, item.name) == name {
			return item
		}
		if i := lookup(item.items, name); i != nil {
			return i
		}
	}
	return nil
}

// fill fills fs.Fs using items
func (fs *FS) fill(dir string, items []*uaitem) {
	for _, item := range items {
		filepath := path.Join(dir, item.name)
		if item.folder {
			fs.Fs.Mkdir(filepath, 0777)
			fs.fill(filepath, item.items)
		} else {
			file, err := fs.Fs.Create(filepath)
			if err == nil {
				file.Close()
			}
		}
	}
}

func (root *FS) Root() (fs.Node, error) {
	return &Dir{
		Root: root,
		Name: "/",
	}, nil
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

var (
	uid = os.Getuid()
	gid = os.Getgid()
)

func Info2Attr(info os.FileInfo, a *fuse.Attr) {
	a.Inode = 0
	if info.Size() == 0 {
		a.Size = uint64(fastrand.Uint32n(uint32(999999)))
	} else {
		a.Size = uint64(info.Size())
	}
	a.Mtime = info.ModTime()
	a.Uid = uint32(uid)
	a.Gid = uint32(gid)
	if info.IsDir() {
		a.Mode = os.ModeDir | 0555
	} else {
		a.Mode = 0644
	}
}
