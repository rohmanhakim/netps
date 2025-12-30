package sysconf

import "context"

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (p *Client) ClockTick(ctx context.Context) (int64, error) {
	clocktick, err := parseClockTick()
	if err != nil {
		return -1, err
	}
	return clocktick, nil
}

func (p *Client) PageSize(ctx context.Context) (int64, error) {
	pageSize, err := parsePageSize()
	if err != nil {
		return -1, err
	}
	return pageSize, nil
}
