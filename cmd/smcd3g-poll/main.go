/*
smcd3g-poll polls an SMC SMCD3G Cable Modem Gateway's cable modem status
and writes it to standard output in a human readable format.
*/
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	modemAddr = "10.1.10.1"
	modemUser = "cusadmin"
	modemPass = "highspeed"

	modemBaseURL = "http://" + modemAddr
	modemLoginURL = modemBaseURL + "/goform/login"
	modemStatusURL = modemBaseURL + "/user/feat-gateway-modem.asp"
)

var statusRE = regexp.MustCompile(`(?m)^var Cm(\w+)Base *= *"([^"]*)";$`)

type status map[string][4]float64

func fields(s []byte) (res [4]float64) {
	for i, f := range bytes.SplitN(s, []byte("|"), 5) {
		if i >= 4 {
			break
		}
		v := string(bytes.TrimSpace(f))
		if v == "" || v == "ERR" {
			continue
		}
		if strings.HasSuffix(v, "QAM") {
			v = strings.TrimSpace(v[:len(v)-3])
		}
		x, err := strconv.ParseFloat(v, 64)
		if err != nil {
			panic(err)
		}
		res[i] = x
	}
	return
}

func scrapeStatus(html []byte) status {
	res := make(status)
	for _, m := range statusRE.FindAllSubmatch(html, -1) {
		res[string(m[1])] = fields(m[2])
	}
	return res
}

func main() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := http.Client{Jar: jar}

	resp1, err := client.PostForm(modemLoginURL, url.Values{"user": {modemUser}, "pws": {modemPass}})
	if err != nil {
		panic(err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != 200 {
		panic("login post failed")
	}

	resp2, err := client.Get(modemStatusURL)
	if err != nil {
		panic(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		panic("status get failed")
	}

	body, err := ioutil.ReadAll(resp2.Body)
	if err != nil {
		panic(err)
	}

	st := scrapeStatus(body)
	fmt.Println("Downstream Channels")
	st.Row("DownstreamFrequency", "Frequency (MHz)", 1)
	st.Row("DownstreamDSLockStatus", "Lock Status", 1)
	st.Row("DownstreamQam", "Modulation (QAM)", 1)
	st.Row("DownstreamChannelPowerdBmV", "Power (dBmV)", 1)
	st.Row("DownstreamSnr", "SNR", 1)
	fmt.Println("")
	fmt.Println("Upstream Channels")
	st.Row("UpstreamFrequency", "Frequency (MHz)", 1e-6)
	st.Row("UpstreamLockStatus", "Lock Status", 1)
	st.Row("UpstreamModu", "Modulation (QAM)", 1)
	st.Row("UpstreamChannelPower", "Power (dBmV)", 1)
	st.Row("UpstreamChannelId", "Channel ID", 1)
}

func (st status) Row(key, label string, scale float64) {
	fmt.Printf("  %-18s", label+":")
	for _, v := range st[key] {
		fmt.Printf("  %12.6g", v * scale)
	}
	fmt.Println()
}
