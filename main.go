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

var maxFiles = flag.Int("n", 5, "Max files")

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("%s <username>@alu.ua.es <mount point>\n", os.Args[0])
		os.Exit(0)
	}
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
	ctx := daemon.Context{
		Env: append(os.Environ(), "psswrd="+pass),
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

	if !strings.Contains(os.Args[1], "@alu") {
		os.Args[1] += "@alu.ua.es"
	}

	client, cookies, err := login(
		os.Args[1], pass,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	created := false
	if _, err := os.Stat(os.Args[2]); err != nil {
		created = true
		os.Mkdir(os.Args[2], 0755)
	}

	root := &FS{
		Cookies: cookies,
		Client:  client,
		Name:    os.Args[1],
		Pass:    pass,
		Fs:      afero.NewMemMapFs(),
		items:   make([]*uaitem, 0),
	}
	root.Fs.MkdirAll("/", 0777)
	root.getFolders()
	for i := range root.items {
		root.getItems(root.items[i])
	}
	root.fill("/", root.items)

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

	gocron.Every(2).Minutes().Do(checkFiles, root)

	err = fs.Serve(fconn, root)
	if err != nil {
		log.Fatal(err)
	}
	if created {
		os.Remove(os.Args[2])
	}
}

func checkFiles(fs *FS) {
	fs.Lock()
	files := fs.downloadFiles
	fs.downloadFiles = fs.downloadFiles[:0]
	fs.Unlock()
	for i := 0; i < len(files); i++ {
		file := files[i]
		st, err := fs.Fs.Stat(file)
		if err != nil {
			continue
		}
		if time.Since(st.ModTime()) < time.Minute*20 {
			fs.Fs.Remove(file)
			file, err := fs.Fs.Create(file)
			if err == nil {
				file.Close()
			}
			files = append(files[:i], files[i+1:]...)
			i--
		}
	}
	fs.Lock()
	fs.downloadFiles = append(fs.downloadFiles, files...)
	fs.Unlock()
}

var _ fs.FS = (*FS)(nil)

type FS struct {
	sync.RWMutex
	downloadFiles []string
	Cookies       *cookiejar.CookieJar
	Client        *fasthttp.Client
	Name          string
	Pass          string
	Fs            afero.Fs
	items         []*uaitem
}

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

func (fs *FS) fill(dir string, items []*uaitem) {
	for _, item := range items {
		filepath := path.Join(dir, item.name)
		if item.folder {
			fs.Fs.Mkdir(filepath, 0777)
		} else {
			file, err := fs.Fs.Create(filepath)
			if err == nil {
				file.Close()
			}
			continue
		}
		fs.fill(filepath, item.items)
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
