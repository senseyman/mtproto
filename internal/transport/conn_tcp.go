package transport

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/xelaj/go-dry/ioutil"
)

type tcpConn struct {
	cancelReader *ioutil.CancelableReader
	conn         *net.TCPConn
	timeout      time.Duration
}

func NewTCP(host string, timeout time.Duration) (Conn, error) {
	return NewTCPWithCtx(context.Background(), host, timeout)
}

func NewTCPWithCtx(ctx context.Context, host string, timeout time.Duration) (Conn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return nil, errors.Wrap(err, "resolving tcp")
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, errors.Wrap(err, "dialing tcp")
	}

	return &tcpConn{
		cancelReader: ioutil.NewCancelableReader(ctx, conn),
		conn:         conn,
		timeout:      timeout,
	}, nil
}

func (t *tcpConn) Close() error {
	return t.conn.Close()
}

func (t *tcpConn) Write(b []byte) (int, error) {
	return t.conn.Write(b)
}

func (t *tcpConn) Read(b []byte) (int, error) {
	if t.timeout > 0 {
		err := t.conn.SetReadDeadline(time.Now().Add(t.timeout))
		check(err)
	}

	n, err := t.cancelReader.Read(b)
	if err != nil {
		if e, ok := err.(*net.OpError); ok {
			if e.Err.Error() == "i/o timeout" {
				// timeout? no worries, but we must reconnect tcp connection
				return 0, errors.Wrap(err, "required to reconnect!")
			}
		}
		switch err {
		case io.EOF, context.Canceled:
			return 0, err
		default:
			return 0, errors.Wrap(err, "unexpected error")
		}
	}
	return n, nil
}
