package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/websocket"
)

var ws *websocket.Conn
var masscan *exec.Cmd
var sourcePort = "61234"
var defaultArgs = []string{
	"--open", "--open-only", "-p5900-5910", "--banners",
	"--source-port", sourcePort, "-oD", "/dev/stdout",
	"--exclude", "0.0.0.0/8", "--exclude", "10.0.0.0/8",
	"--exclude", "100.64.0.0/10", "--exclude", "127.0.0.0/8",
	"--exclude", "169.254.0.0/16", "--exclude", "172.16.0.0/12",
	"--exclude", "192.0.0.0/24", "--exclude", "192.0.2.0/24",
	"--exclude", "192.88.99.0/24", "--exclude", "192.168.0.0/16",
	"--exclude", "192.18.0.0/15", "--exclude", "198.51.100.0/24",
	"--exclude", "203.0.113.0/24", "--exclude", "224.0.0.0/4",
	"--exclude", "233.252.0.0/24", "--exclude", "240.0.0.0/4",
	"--exclude", "255.255.255.255/32",
	"--exclude", "6.0.0.0/7", "--exclude", "9.0.0.0/8",
	"--exclude", "10.0.0.0/7", "--exclude", "19.0.0.0/8",
	"--exclude", "21.0.0.0/7", "--exclude", "25.0.0.0/7",
	"--exclude", "28.0.0.0/8", "--exclude", "29.0.0.0/7",
	"--exclude", "33.0.0.0/8", "--exclude", "48.0.0.0/8",
	"--exclude", "53.0.0.0/8", "--exclude", "55.0.0.0/7",
	"--exclude", "214.0.0.0/7","-n",
}
var status = ""
var server = "localhost:8080"
var password = "DinoKhoa1234"
var started = false
var rate = ""

func main() {
	user, err := user.Current()
	if err != nil || user.Uid != "0" {
		log.Fatalln("Run as root!")
	}

	if len(os.Args) > 1 {
		rate = os.Args[1]
	} else {
		rate = "10000000"
	}

	iptables := exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", sourcePort, "-j", "DROP")
	if err := iptables.Run(); err != nil {
		log.Println(err)
		log.Println("Please install iptables to work propery")
	}

	sec := ""
	if len(os.Args) > 2 && os.Args[2] == "http" {
		sec = ""
	}

	auth := base64.StdEncoding.EncodeToString([]byte("client:" + password))

	ws, err = websocket.DialConfig(&websocket.Config{
		Location: &url.URL{Scheme: "ws" + sec, Host: server, Path: "/api/client"},
		Origin: &url.URL{Scheme: "http", Host: server},
		Version: websocket.ProtocolVersionHybi13,
		Header: http.Header{"Authorization": {"Basic " + auth}},
	})
	if err != nil {
		log.Fatalln(err)
	}
	defer ws.Close()
	log.Println("Connected")

	for {
		msg := readMSG()
		if len(msg) < 1 {
			continue
		}
		log.Println("Got", msg)

		switch msg[0] {
		case "status": writeMSG("status", getStatus())
		case "start": writeMSG("start", start())
		case "stop": writeMSG("stop", stop())
		case "range": go scanRange(msg[1])
		case "vnc": log.Println(msg[1])
		case "ping": writeMSG("pong")
		}
	}
}

func start() string {
	if started || running() {
		return "Already started"
	}
	started = true
	writeMSG("range")
	return "Started successfully"
}

func stop() string {
	started = false
	if !running() {
		return "Already stopped"
	}
	err := masscan.Process.Kill()
	if err != nil {
		return err.Error()
	}
	return "Stopped successfully"
}

func getStatus() string {
	if running() {
		return strings.TrimSpace(status)
	}
	return "Idling"
}

func scanRange(rnge string) {
	if rnge == "stop" {
		stop()
		return
	}
	if running() {
		log.Printf("Got range %s even though masscan still running", rnge)
		return
	}
	if !started {
		log.Printf("Got range %s even though should stop", rnge)
		return
	}
	status = "Starting..."
	args := append(defaultArgs, "--rate", rate, rnge)
	log.Println("Running masscan with args", args)
	masscan = exec.Command("masscan", args...)
	stdout, err := masscan.StdoutPipe()
	if err != nil {
		log.Fatalln(err)
	}
	stderr, err := masscan.StderrPipe()
	if err != nil {
		log.Fatalln(err)
	}
	err = masscan.Start()
	if err != nil {
		log.Fatalln(err)
	}

	go readStatus(stderr)
	readVNCs(stdout)
	masscan.Wait()
	if started {
		writeMSG("range")
	}
}

func readStatus(from io.ReadCloser) {
	scanner := bufio.NewScanner(from)
	scanner.Split(scanStatus)
	r, err := regexp.Compile(`waiting -[0-9]+-secs`)
	if err != nil {
		log.Fatalln(err)
	}
	for scanner.Scan() {
		status = scanner.Text()
		if r.MatchString(status) {
			if running() {
				masscan.Process.Kill()
			}
			break
		}
		fmt.Fprint(os.Stderr, status)
	}
}

func scanStatus(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if data[len(data) - 1] == '\n' {
		return len(data), nil, nil
	}

	if data[len(data) - 1] == '\r' {
		return len(data), data, nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return
}

func readVNCs(from io.ReadCloser) {
	scanner := bufio.NewScanner(from)
	for scanner.Scan() {
		var data map[string]any
		err := json.Unmarshal([]byte(scanner.Text()), &data)
		if err != nil {
			log.Println("Error parsing json", scanner.Text())
			break
		}
		d := data["data"].(map[string]any)
		if data["rec_type"] != "banner" || d["service_name"] != "vnc" {
			continue
		}
		log.Println("Putting in", data["ip"], data["port"])
		writeMSG("vnc", data["ip"].(string), strconv.Itoa(int(data["port"].(float64))))
	}
}

func running() bool {
	return masscan != nil && masscan.ProcessState == nil
}

func readMSG() []string {
	buf := make([]byte, 1024)
	n, err := ws.Read(buf)
	if err != nil {
		log.Fatalln(err)
	}
	var res []string
	err = json.Unmarshal(buf[:n], &res)
	if err != nil {
		log.Fatalln(err)
	}
	return res
}

func writeMSG(msg ...string) {
	b, err := json.Marshal(msg)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = ws.Write(b)
	if err != nil {
		log.Fatalln(err)
	}
}
