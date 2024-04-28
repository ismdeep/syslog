package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"
)

var currentFileName string
var output *os.File

func GetFileName() string {
	return fmt.Sprintf("%v.log", time.Now().Format("2006-01-02"))
}

func StartCompressOldFilesCron() {
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("recovered in StartCompressOldFilesCron, recover: %v\n", r)
			}
		}()
		for i := 2; i < 30; i++ {
			fileName := fmt.Sprintf("%v.log", time.Now().Add(time.Duration(-1*i*24)*time.Hour).Format("2006-01-02"))
			stat, err := os.Stat(fileName)
			if err == nil && !stat.IsDir() {
				cmd := exec.Command("xz", "-T", "0", fileName)
				cmd.Stdin = nil
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("Failed to compress %v: %v\n", fileName, err)
				}
			}
		}
	}

	go func() {
		for {
			run()
			time.Sleep(5 * time.Minute)
		}
	}()
}

func UpdateOutputInstance(fileName string) {
	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		fmt.Printf("[WARN] failed to open output file: %v\n", err)
		return
	}
	output = f
	currentFileName = fileName
}

func init() {
	UpdateOutputInstance(GetFileName())

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			fileName := GetFileName()
			if currentFileName != fileName {
				UpdateOutputInstance(fileName)
			}
		}
	}()
}

func handleSyslogMessages(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("[WARN] failed to close connection local_addr: [%v] remote_addr: [%v]. err: %v\n",
				conn.LocalAddr(),
				conn.RemoteAddr(),
				err)
		}
	}()

	r := bufio.NewReader(conn)

	for {
		s, err := r.ReadString(byte('\n'))
		if err != nil {
			fmt.Printf("[WARN] failed to read from connection local_addr: [%v] remote_addr: [%v]. err: %v\n",
				conn.LocalAddr().String(),
				conn.RemoteAddr().String(),
				err)
			break
		}

		_, _ = output.WriteString(s)
	}
}

func main() {
	StartCompressOldFilesCron()

	// 启动服务
	listener, err := net.Listen("tcp", "0.0.0.0:5140") // 使用 TCP 协议监听 5140 端口
	if err != nil {
		log.Fatal("Error starting syslog server:", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Printf("[WARN] failed to close listener: %v\n", err)
		}
	}()
	log.Println("Syslog server started on 0.0.0.0:5140")

	// 等待并处理连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleSyslogMessages(conn)
	}
}
