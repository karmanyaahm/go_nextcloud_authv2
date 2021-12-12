package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type answer struct {
	Server      string
	LoginName   string
	AppPassword string
	err         error
}

type initiateLogin struct {
	Poll struct {
		Token    string
		Endpoint string
	}
	Login string
}

func initLogin(ctx context.Context, ans chan answer, server string, ua string) initiateLogin {
	var i initiateLogin
	serverAddr, err := url.Parse(server)
	errFatal(ans, err)

	serverAddr.Path += "/index.php/login/v2"
	if serverAddr.Scheme == "" {
		serverAddr.Scheme = "http"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverAddr.String(), bytes.NewBufferString(""))
	errFatal(ans, err)
	req.Header.Set("User-Agent", ua)
	resp, err := http.DefaultClient.Do(req)
	errFatal(ans, err)

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errFatal(ans, errors.New("Nextcloud server login page not OK - "+strconv.Itoa(resp.StatusCode)+" - "+resp.Status))
	}

	b, err := io.ReadAll(resp.Body)
	errFatal(ans, err)
	err = json.Unmarshal(b, &i)
	errFatal(ans, err)

	return i
}
func errFatal(ans chan answer, err error) {
	if err != nil {
		ans <- answer{err: err}
	}
}

//
// user agent like Golang_Example_Nextcloud_login/1.0
//Note: this blocks stdin until the program exits. It's meant to be used in a dedicated 'login' command.
func Authenticate(ctx context.Context, server string, useragent string, stdout io.Writer, stdin io.Reader) (prefServer, loginName, appPassword string, err error) {
	ans := make(chan answer)
	pingAgain := make(chan struct{})
	ctx, ctxCancel := context.WithCancel(ctx)

	fmt.Fprintln(stdout, "Press Ctrl+C to exit ...")
	fmt.Fprintln(stdout, "Initiating login...")
	i := initLogin(ctx, ans, server, useragent)

	fmt.Fprintf(stdout, "Open this link in your browser and log in: %s\n", i.Login)
	fmt.Fprintln(stdout, "Press Enter to check for login again ...")
	go pingLoop(ctx, i, ans, pingAgain, server, stdout)
	go keyListener(ctx, pingAgain, stdin)

	finalAnswer := <-ans
	ctxCancel()
	return finalAnswer.Server, finalAnswer.LoginName, finalAnswer.AppPassword, finalAnswer.err
}

func pingLoop(ctx context.Context, i initiateLogin, ans chan answer, pingAgain chan struct{}, server string, stdout io.Writer) {
	lastPingEnd := time.Unix(0, 0)
	pingIntervalSecond := 5.0
	const minPingInterval = 1 * time.Second

	for {
		select {
		case <-pingAgain:
		case <-time.After(time.Duration(pingIntervalSecond) * time.Second):
			pingIntervalSecond = math.Log2(math.Pow(2, pingIntervalSecond/5)+1) * 5
		case <-ctx.Done():
			return
		}
		if lastPingEnd.Add(minPingInterval).Before(time.Now()) {
			fmt.Fprintln(stdout, "pinging now...")
			ping(i, ans, server)
			lastPingEnd = time.Now()
			//fmt.Println(pingIntervalSecond)
		}

	}
}

func ping(i initiateLogin, ans chan answer, server string) {
	resp, err := http.Post(i.Poll.Endpoint, "application/x-www-form-urlencoded", strings.NewReader("token="+i.Poll.Token))
	defer resp.Body.Close()
	if err != nil {
		ans <- answer{err: err}
	}

	if resp.StatusCode == 200 {
		var answ answer
		b, err := io.ReadAll(resp.Body)
		errFatal(ans, err)

		err = json.Unmarshal(b, &answ)
		errFatal(ans, err)

		ans <- answ
	} else if resp.StatusCode == 404 {
		//do nothing, let it poll more
	} else {
		errFatal(ans, errors.New("Nextcloud polling error"))
	}
}

func keyListener(ctx context.Context, pingAgain chan struct{}, stdin io.Reader) {
	var s string
	for {
		_, e := fmt.Fscanln(stdin, &s)
		if e == io.EOF {
			break
		}
		s = strings.TrimSpace(s)

		if s == "cmd" {
			//TODO insert possible commands here???
		} else {
			pingAgain <- struct{}{}
		}
	}
}
