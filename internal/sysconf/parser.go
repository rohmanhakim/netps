package sysconf

import (
	"github.com/tklauser/go-sysconf"
)

func parseClockTick() (int64, error) {
	clktck, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
	if err == nil {
		return clktck, nil
	} else {
		return -1, err
	}
}

func parsePageSize() (int64, error) {
	ps, err := sysconf.Sysconf(sysconf.SC_PAGE_SIZE)
	if err != nil {
		return 0, err
	}
	return ps, nil
}
