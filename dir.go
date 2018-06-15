package main

import (
	"path"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type Dir struct {
	Root *FS
	Name string
	item *uaitem
}

var _ fs.Node = (*Dir)(nil)

// Attr fills a with file attributes. Ignores context.
func (d *Dir) Attr(_ context.Context, a *fuse.Attr) error {
	st, err := d.Root.Fs.Stat(d.Name)
	if err != nil {
		return fuse.ENOENT
	}
	Info2Attr(st, a)
	return nil
}

var _ fs.NodeGetxattrer = (*Dir)(nil)

func (d *Dir) Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error {
	return fuse.ErrNoXattr
}

var _ fs.NodeStringLookuper = (*Dir)(nil)

// Lookup search file inside directory
func (d *Dir) Lookup(_ context.Context, name string) (fs.Node, error) {
	f, err := d.Root.Fs.Open(d.Name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	files, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.Name() == name {
			if f.IsDir() {
				dir := &Dir{
					Name: path.Join(d.Name, name),
					Root: d.Root,
				}
				dir.Root.Fs.Mkdir(dir.Name, 0777)
				return dir, nil
			}
			file := &File{
				Root: d.Root,
				Name: path.Join(d.Name, name),
			}
			ff, err := d.Root.Fs.Create(file.Name)
			if ff == nil {
				ff.Close()
			}
			return file, err
		}
	}
	return nil, fuse.ENOENT
}

var _ fs.HandleReadDirAller = (*Dir)(nil)

// ReadDirAll returns all files in directory
func (d *Dir) ReadDirAll(_ context.Context) ([]fuse.Dirent, error) {
	file, err := d.Root.Fs.Open(d.Name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	files, err := file.Readdir(0)
	if err != nil {
		return nil, err
	}
	var fd = make([]fuse.Dirent, 0, len(files))
	for _, f := range files {
		t := fuse.DT_File
		if f.IsDir() {
			t = fuse.DT_Dir
		}
		fd = append(fd, fuse.Dirent{
			Name: f.Name(),
			Type: t,
		})
	}
	return fd, nil
}
