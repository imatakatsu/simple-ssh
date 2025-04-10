package main

import (
	"fmt"
	"time"

	ssh "github.com/imatakatsu/simple-ssh"
)

var srv ssh.Serv

func main() {
	err := srv.Init(terminal)
	if err != nil {
		fmt.Println(err)
		return
	}
	srv.Listen(":2222")
}

func terminal(conn ssh.SshConn) {
	defer conn.Close()
	conn.Writeln("\x1bcWelcome to Simple SSH Example!!! wooow")
	conn.Writef("here u can write (some data: %v) formatted strings!!\r\n", time.Now())
	conn.Write("\r\nit`s an ssh echo server, lolll\r\n\r\n")
	for {
		ans, err := conn.Readline()
		if err != nil {
			fmt.Println(err)
			return
		}
		if ans == "exit" || ans == "q" {
			conn.Writeln("byee")
			return
		}
		conn.Writeln("you wrote:", ans)
	}
}
