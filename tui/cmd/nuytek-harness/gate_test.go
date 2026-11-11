package main

import "testing"

func TestTerminalGate(t *testing.T) {
	cases := []struct {
		name          string
		stdin, stdout bool
		want          bool
	}{
		{"neither is a terminal", false, false, false},
		{"only stdin is a terminal", true, false, false},
		{"only stdout is a terminal", false, true, false},
		{"both are terminals", true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			queried := map[int]bool{}
			isTerminal := func(fd int) bool {
				queried[fd] = true
				return map[int]bool{stdinFd: c.stdin, stdoutFd: c.stdout}[fd]
			}
			if got := terminalGate(isTerminal); got != c.want {
				t.Errorf("terminalGate = %v, want %v", got, c.want)
			}
			if !queried[stdinFd] || !queried[stdoutFd] {
				t.Errorf("fds queried = %v, want both %d and %d", queried, stdinFd, stdoutFd)
			}
		})
	}
}
