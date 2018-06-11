package main

import (
	"bytes"
	"fmt"

	"github.com/erikdubbelboer/fasthttp"
	"github.com/themester/fcookiejar"
)

func doReqFollowRedirects(
	req *fasthttp.Request, res *fasthttp.Response,
	client *fasthttp.Client, cookies *cookiejar.CookieJar) (err error) {

	var url, body []byte
	var status int
	var referer string
	for {
		// use compression!!!11!
		// compression is better. Compress your life :')
		req.Header.Add("Accept-Encoding", "gzip")

		if referer != "" {
			req.Header.Add("Referer", referer)
		}
		// writting cookies to request
		cookies.AddToRequest(req)

		err = client.Do(req, res)
		if err != nil {
			goto end
		}
		// reading cookies from the response
		cookies.ResponseCookies(res)

		status = res.Header.StatusCode()
		if status < 300 || status > 399 { // no redirect
			break
		}
		referer, url = string(url), res.Header.Peek("Location")
		if len(url) == 0 {
			err = fmt.Errorf("Status code is redirect (%d) but no one location have been provided", status)
			goto end
		}
		req.Reset()
		res.Reset()

		req.SetRequestURIBytes(url)
	}
	switch status {
	case fasthttp.StatusOK:
	case fasthttp.StatusUnauthorized:
		err = fmt.Errorf("Incorrect password")
		goto end
	default:
		err = fmt.Errorf("error server returned status code %d", status)
		goto end
	}
	body = res.Body()
	if bytes.Equal(res.Header.Peek("Content-Encoding"), []byte("gzip")) {
		// gunzipping
		_, err = fasthttp.AppendGunzipBytes(body, body)
		if err != nil {
			panic(err)
		}
		res.SetBody(body)
	}
end:
	return err
}
