package main

import (
	"bytes"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/erikdubbelboer/fasthttp"
)

type uaitem struct {
	cod     string
	codasig string
	name    string
	path    string
	items   []*uaitem
	folder  bool
}

var (
	rcod  = regexp.MustCompile(`data-codasi="(.*?)"`)
	rasig = regexp.MustCompile(`<span class="asi">(.*?)</span>`)
	rfold = regexp.MustCompile(`<div class="(.*?)" data-id`)
)

func (fs *FS) getFolders() error {
	client := fs.Client
	cookies := fs.Cookies
	args := fasthttp.AcquireArgs()
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseArgs(args)
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.SetRequestURI(urlFuck)
	err := doReqFollowRedirects(req, res, client, cookies)
	if err != nil {
		return err
	}
	req.Reset()
	res.Reset()

	args.Set("codasi", "-1")
	args.Set("direccion", "")
	args.Set("expresion", "")
	args.Set("filtro", "")
	args.Set("idmat", "-1")
	args.Set("pendientes", "N")

	args.WriteTo(req.BodyWriter())

	// preparing request
	req.SetRequestURI(urlFolders)
	req.Header.SetMethod("POST")

	err = doReqFollowRedirects(req, res, client, cookies)
	if err != nil {
		return err
	}

	body := res.Body()
	codeMatch := rcod.FindAllSubmatch(body, -1)
	nameMatch := rasig.FindAllSubmatch(body, -1)
	foldMatch := rfold.FindAllSubmatch(body, -1)
	// ignoring first
	for i := 0; i < len(codeMatch) && i < len(nameMatch); i++ {
		for j := 1; j < len(codeMatch[i]) && j < len(nameMatch[i]); j += 2 {
			name := formatName(string(nameMatch[i][j]))
			it := &uaitem{
				folder:  strings.Contains(string(foldMatch[i][j]), "carpeta"),
				cod:     "-1",
				path:    path.Join("/", name),
				codasig: string(codeMatch[i+1][j]),
				name:    name,
			}
			fs.items = append(fs.items, it)
		}
	}

	return nil
}

var htmlescapecodes = []string{
	"&#191;", "a",
	"&#193;", "A",
	"&#241;", "ny",
	"&#225;", "a",
	"&#233;", "e",
	"&#237;", "i",
	"&#243;", "o",
	"&#250;", "u",
	"&#209;", "NY",
	"&#193;", "A",
	"&#201;", "E",
	"&#205;", "i",
	"&#211;", "O",
	"&#218;", "U",
}

var replacer = strings.NewReplacer(htmlescapecodes...)

func formatName(s string) string {
	return replacer.Replace(s)
}

var (
	rname = regexp.MustCompile(`class="nombre" >(.*?)</span>`)
	rid   = regexp.MustCompile(`<div class="columna1">(.*?)</div>`)
)

func (fs *FS) download(item *uaitem) error {
	client := fs.Client
	cookies := fs.Cookies
	args := fasthttp.AcquireArgs()
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()

	args.Set("identificadores", item.cod)
	args.Set("codasis", item.codasig)

	req.SetRequestURI(urlDownload)
	req.Header.SetContentType("application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.SetMethod("POST")

	args.WriteTo(req.BodyWriter())
	fasthttp.ReleaseArgs(args)

	to := formatName(path.Join(item.path, item.name))

	err := doReqFollowRedirects(req, res, client, cookies)
	if err != nil {
		return err
	}
	if bytes.Equal(res.Header.ContentType(), []byte("application/zip")) {
		if !strings.Contains(
			path.Ext(to), ".zip",
		) {
			to += ".zip"
		}
	}

	file, err := fs.Fs.OpenFile(to, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	res.BodyWriteTo(file)
	file.Close()

	fs.Lock()
	fs.downloadFiles = append(fs.downloadFiles, to)
	fs.Unlock()

	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(res)
	return nil
}

func (fs *FS) getItems(item *uaitem) error {
	client := fs.Client
	cookies := fs.Cookies

	dirs := make([]*uaitem, 1)
	dirs[0] = item
	for inc := 0; inc < len(dirs); inc++ {
		args := fasthttp.AcquireArgs()
		req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()

		args.Set("idmat", dirs[inc].cod)
		args.Set("codasi", item.codasig)
		args.Set("expresion", "")
		args.Set("direccion", "")
		args.Set("filtro", "")
		args.Set("pendientes", "N")
		args.Set("fechadesde", "")
		args.Set("fechahasta", "")
		args.Set("busquedarapida", "N")
		args.Set("idgrupo", "")

		args.WriteTo(req.BodyWriter())

		req.Header.SetContentType("application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.SetMethod("POST")
		req.SetRequestURI(urlFiles)

		err := doReqFollowRedirects(req, res, client, cookies)
		if err != nil {
			return err
		}
		body := res.Body()

		fasthttp.ReleaseArgs(args)
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)

		idMatch := rid.FindAllSubmatch(body, -1)
		nameMatch := rname.FindAllSubmatch(body, -1)
		dirMatch := rfold.FindAllSubmatch(body, -1)
		for i := 0; i < len(idMatch); i++ {
		sloop:
			for j := 1; j < len(idMatch[i]); j += 2 {
				folder := strings.Contains(string(dirMatch[i][j]), "carpeta")
				if !folder {
					if !strings.Contains(string(dirMatch[i][j]), "archivo") {
						continue sloop
					}
				}
				cod := string(idMatch[i][j])
				name := string(nameMatch[i][j])
				it := &uaitem{
					cod:     cod,
					codasig: dirs[inc].codasig,
					path:    dirs[inc].path,
					name:    formatName(name),
					folder:  folder,
				}
				if folder {
					it.path = path.Join(it.path, it.name)
					dirs = append(dirs, it)
				}
				dirs[inc].items = append(dirs[inc].items, it)
			}
		}

		// deleting processed dir
		dirs = dirs[1:]
		inc--
	}
	return nil
}
