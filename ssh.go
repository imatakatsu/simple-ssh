package simple_ssh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Serv struct {
	KeyFile  string
	Config   *ssh.ServerConfig
	Listener *net.Listener
	Handler  ServHandler
}

type ServHandler func(c SshConn)

func (s *Serv) Init(handler ServHandler) error {
	var privKey ssh.Signer
	if s.KeyFile == "" {
		s.KeyFile = ".privKey"
	}
	if _, err := os.Stat(s.KeyFile); os.IsNotExist(err) {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errors.New("failed to generate rsa key")
		}
		pb := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		}
		err = os.WriteFile(s.KeyFile, pem.EncodeToMemory(pb), 0600)
		if err != nil {
			return errors.New("failed to write key to file")
		}
		privKey, err = ssh.NewSignerFromKey(rsaKey)
		if err != nil {
			return errors.New("failed to create signer")
		}
	} else {
		kb, err := os.ReadFile(s.KeyFile)
		if err != nil {
			os.Remove(s.KeyFile)
			return errors.New("failed to open key file")
		}
		privKey, err = ssh.ParsePrivateKey(kb)
		if err != nil {
			os.Remove(s.KeyFile)
			return errors.New("failed to parse private key, restart script")
		}
	}
	s.Config = &ssh.ServerConfig{
		NoClientAuth: true,
	}
	s.Config.AddHostKey(privKey)
	s.Handler = handler
	return nil
}

func (s *Serv) Listen(host string) error {
	l, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go func(conn net.Conn, s *Serv) {
			defer conn.Close()
			sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.Config)
			if err != nil {
				return
			}
			defer sshConn.Close()
			go ssh.DiscardRequests(reqs)
			for nc := range chans {
				if nc.ChannelType() != "session" {
					nc.Reject(ssh.UnknownChannelType, "unknown channel type")
					continue
				}
				c, r, err := nc.Accept()
				if err != nil {
					return
				}
				defer c.Close()
				go func() {
					for req := range r {
						if req.Type == "shell" {
							req.Reply(true, nil)
							s.Handler(SshConn{conn: c})
						} else {
							req.Reply(false, nil)
						}
					}
				}()
			}
		}(conn, s)
	}
}

type SshConn struct {
	conn ssh.Channel
}

func (s *SshConn) Close() error {
	return s.conn.Close()
}

func (s *SshConn) Write(a ...any) error {
	msg := fmt.Sprint(a...)
	if _, err := s.conn.Write([]byte(msg)); err != nil {
		return err
	}
	return nil
}

func (s *SshConn) Writeln(a ...any) error {
	msg := fmt.Sprintln(a...)
	if _, err := s.conn.Write([]byte(msg)); err != nil {
		return err
	}
	return nil
}

func (s *SshConn) Writef(f string, a ...any) error {
	msg := fmt.Sprintf(f, a...)
	if _, err := s.conn.Write([]byte(msg)); err != nil {
		return err
	}
	return nil
}

func (s *SshConn) Readline() (string, error) {
	var gbuf bytes.Buffer
	for !strings.HasSuffix(gbuf.String(), "\n") {
		buf := make([]byte, 1)
		_, err := s.conn.Read(buf)
		if err != nil {
			return gbuf.String(), err
		}
		gbuf.WriteByte(buf[0])
	}
	return strings.TrimSpace(gbuf.String()), nil
}
