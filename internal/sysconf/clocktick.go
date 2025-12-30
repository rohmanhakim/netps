package sysconf

import (
	"github.com/tklauser/go-sysconf"
)

func Get() (int64, error) {
	clktck, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err == nil {
		return clktck, nil
	} else {
		return -1, err
	}
}
