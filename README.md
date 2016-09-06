# go-sshd [![Build Status](https://travis-ci.org/hnakamur/go-sshd.svg?branch=master)](https://travis-ci.org/hnakamur/go-sshd) [![Go Report Card](https://goreportcard.com/badge/github.com/hnakamur/go-sshd)](https://goreportcard.com/report/github.com/hnakamur/go-sshd) [![GoDoc](https://godoc.org/github.com/hnakamur/go-sshd?status.svg)](https://godoc.org/github.com/hnakamur/go-sshd) [![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/hyperium/hyper/master/LICENSE)

A sshd written in Go. Forked from [github.com/jpillora/go-and-ssh](https://github.com/jpillora/go-and-ssh).
Only supported request types are exec, shell, pty-req, and window-change.

## Example usage

### Run the example server.

Get the source.

```
go get github.com/hnakamur/go-sshd
cd $GOPATH/src/github.com/hnakamur/go-sshd/example
```

Generate the sever host key.

```
ssh-keygen -t rsa -b 2048 -f id_rsa -N ''
```

Run the server at the address 127.0.0.1:2022

```
go run main.go
```

### Run a ssh client

Run a ssh client. You can see the password in the example source.

```
$ ssh -o UserKnownHostsFile=/dev/null -p 2022 foo@127.0.0.1
The authenticity of host '[127.0.0.1]:2022 ([127.0.0.1]:2022)' can't be established.
RSA key fingerprint is SHA256:wr...(masked)...wc.
Are you sure you want to continue connecting (yes/no)? yes
Warning: Permanently added '[127.0.0.1]:2022' (RSA) to the list of known hosts.
foo@127.0.0.1's password:
$ ls
id_rsa  id_rsa.pub  main.go
$ exit
exit
Connection to 127.0.0.1 closed.
```

# License
MIT
