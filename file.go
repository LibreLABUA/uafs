package main

import (
	"context"
	"io"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/spf13/afero"
)

const byteSize = 1024

var bytePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, byteSize)
	},
}

type File struct {
	Name string
	Root *FS
	file afero.File
	item *uaitem
}

// Attr writes file attributes to attr
func (f *File) Attr(_ context.Context, attr *fuse.Attr) error {
	st, err := f.Root.Fs.Stat(f.Name)
	if err != nil {
		return fuse.ENOENT
	}
	Info2Attr(st, attr)
	return nil
}

func (f *File) Open(_ context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	err := f.fill()
	if err != nil {
		return nil, err
	}
	if f.file != nil {
		f.file.Close()
		f.file = nil
	}
	f.file, err = f.Root.Fs.OpenFile(f.Name, int(req.Flags), 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *File) fill() error {
	if f.item == nil {
		f.item = lookup(f.Root.items, f.Name)
		if f.item == nil {
			panic("not found")
		}
	}
	st, err := f.Root.Fs.Stat(f.Name)
	if err != nil {
		return f.Root.download(f.item)
	}
	if st.Size() == 0 || (st.IsDir() && st.Size() == 64) {
		return f.Root.download(f.item)
	}
	return nil
}

func (f *File) ReadAll(_ context.Context) ([]byte, error) {
	err := f.fill()
	if err != nil {
		return nil, err
	}
	bf := bytePool.Get().([]byte)
	b := bytePool.Get().([]byte)

	var n int
	nn := 0
	if f.file != nil {
		f.file.Close()
	}
	f.file, err = f.Root.Fs.Open(f.Name)
	if err != nil {
		goto end
	}
	for {
		n, err = f.file.Read(b)
		if err != nil {
			break
		}
		bf = append(bf[:nn], b[:n]...)
		nn += n
	}
end:
	if err == io.EOF {
		err = nil
	}
	f.file.Close()
	f.file = nil
	bytePool.Put(b)
	return bf[:nn], err
}

func (f *File) Read(_ context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) (err error) {
	var n int
	if f.file == nil {
		err = fuse.ENOTSUP
		goto end
	}
	n, err = f.file.ReadAt(resp.Data[:req.Size], req.Offset)
	resp.Data = resp.Data[:n]
end:
	if err == io.EOF {
		err = nil
	}
	return
}

func (f *File) Release(_ context.Context, req *fuse.ReleaseRequest) error {
	if f.file != nil {
		f.file.Close()
		f.file = nil
	}
	return nil
}

func (f *File) Write(_ context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) (err error) {
	var n int
	if f.file == nil {
		err = fuse.ENOTSUP
		goto end
	}
	n, err = f.file.WriteAt(req.Data, req.Offset)
	resp.Size = n
end:
	return err
}

func (f *File) Getattr(_ context.Context, req *fuse.GetattrRequest, res *fuse.GetattrResponse) error {
	st, err := f.Root.Fs.Stat(f.Name)
	if err != nil {
		return fuse.ENOENT
	}
	Info2Attr(st, &res.Attr)
	return nil
}