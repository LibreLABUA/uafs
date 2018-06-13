package main

import (
	"fmt"
	"regexp"

	"github.com/erikdubbelboer/fasthttp"
	"github.com/themester/fcookiejar"
)

var (
	MB = 1024 * 1024 * 1024
	// regex execution parameter
	regexep = regexp.MustCompile(`name="execution"\svalue="(.*?)"`)
)

func login(user, pass string) (*fasthttp.Client, *cookiejar.CookieJar, error) {
	client := &fasthttp.Client{
		// nice user agent you can choose whatever you want ex: Jomoza sube minecraft
		Name:                "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36",
		MaxResponseBodySize: MB * 100, // 100 mb of download files
	}

	// Be patient... UA's webpage is written in C#
	status, body, err := client.Get(nil, urlNormal)
	if err != nil {
		return nil, nil, err
	}
	if status != 200 {
		return nil, nil, fmt.Errorf("Unexpected response code: %d<>200", status)
	}

	// getting execution parameter
	execp := regexep.FindSubmatch(body)
	if len(execp) == 0 {
		return nil, nil, fmt.Errorf("error getting login parameters...")
	}

	// reusing body as much as possible
	// submatch is the latest parameter so we take len-1
	body = append(body[:0], execp[len(execp)-1]...)

	// getting request, response and arguments for post request
	args := fasthttp.AcquireArgs()
	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseArgs(args)
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	req.Header.SetContentType("application/x-www-form-urlencoded")
	req.Header.SetMethod("POST")

	req.SetRequestURI(urlLogin)

	// setting parameters for post request
	args.Set("_eventId", "submit")
	args.Set("username", user)
	args.Set("password", pass)
	args.Set("geolocation", "")
	args.SetBytesV("execution", body)

	// writting post arguments to request body
	args.WriteTo(req.BodyWriter())

	// creating cookieJar object
	cookies := cookiejar.AcquireCookieJar()

	// make requests xd
	err = doReqFollowRedirects(req, res, client, cookies)
	if err != nil {
		return nil, nil, err
	}

	// getting new cookies
	cookies.ResponseCookies(res)

	// 401 code if password is incorrect
	if status == 401 {
		err = fmt.Errorf("Incorrect password")
	}

	return client, cookies, err
}
