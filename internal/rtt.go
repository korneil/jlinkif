package internal

import (
	"context"
	"net"
	"time"
)

type RTT struct {
	data chan []byte
	done chan struct{}
	conn net.Conn
}

func (r *RTT) Close() {
	if r.conn != nil {
		_ = r.conn.Close()
	}
}

func NewRTT(ctx context.Context, address string) *RTT {
	rtt := &RTT{
		data: make(chan []byte),
	}

	go func() {
		var err error

		reconnectWait := 0

		for ctx.Err() == nil {
			time.Sleep(time.Duration(reconnectWait) * time.Millisecond)
			reconnectWait = (reconnectWait + 64) * 2

			rtt.conn, err = net.Dial("tcp", address)
			if err != nil {
				continue
			}

			reconnectWait = 0

			r := NewLineReader(rtt.conn)
			var line []byte
			for ctx.Err() == nil {
				line, err = r.ReadLine(ctx)
				if line != nil {
					rtt.data <- line
				}
				if err != nil {
					break
				}
			}
			if ctx.Err() != nil {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
	}()

	return rtt
}
