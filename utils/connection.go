package utils

import (
	"github.com/pkg/errors"
	"io"
	"net"
	"time"
)

type TCPListener struct {
	conn                 net.Conn
	planedDisconnect     func(*TCPListener)
	unexpectedDisconnect func(*TCPListener, error)
}

var ErrConnectionClosed = errors.New("connection closed")

// PlanedDisconnect 预料中的断开连接
// 如调用 Close() Connect()
func (t *TCPListener) PlanedDisconnect(f func(*TCPListener)) {
	t.planedDisconnect = f
}

// UnexpectedDisconnect 未预料钟的断开连接
func (t *TCPListener) UnexpectedDisconnect(f func(*TCPListener, error)) {
	t.unexpectedDisconnect = f
}

func (t *TCPListener) Connect(addr *net.TCPAddr) error {
	t.Close()
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return errors.Wrap(err, "dial tcp error")
	}
	t.conn = conn
	return nil
}

func (t *TCPListener) Write(buf []byte) error {
	if t.conn == nil {
		return ErrConnectionClosed
	}
	_, err := t.conn.Write(buf)
	if err != nil {
		if t.conn != nil {
			t.close()
			t.invokeUnexpectedDisconnect(err)
		}
		return ErrConnectionClosed
	}
	return nil
}

func (t *TCPListener) ReadBytes(len int) ([]byte, error) {
	if t.conn == nil {
		return nil, ErrConnectionClosed
	}
	buf := make([]byte, len)
	_, err := io.ReadFull(t.conn, buf)
	if err != nil {
		time.Sleep(time.Millisecond * 100) // 服务器会发送offline包后立即断开连接, 此时还没解析
		if t.conn != nil {
			t.close()
			t.invokeUnexpectedDisconnect(err)
		}
		return nil, ErrConnectionClosed
	}
	return buf, nil
}

func (t *TCPListener) ReadInt32() (int32, error) {
	b, err := t.ReadBytes(4)
	if err != nil {
		return 0, err
	}
	return (int32(b[0]) << 24) | (int32(b[1]) << 16) | (int32(b[2]) << 8) | int32(b[3]), nil
}

func (t *TCPListener) Close() {
	if t.conn == nil {
		return
	}
	t.close()
	t.invokePlanedDisconnect()
}

func (t *TCPListener) close() {
	if t.conn == nil {
		return
	}
	_ = t.conn.Close()
	t.conn = nil
}

func (t *TCPListener) invokePlanedDisconnect() {
	if t.planedDisconnect != nil {
		t.planedDisconnect(t)
	}
}

func (t *TCPListener) invokeUnexpectedDisconnect(err error) {
	if t.unexpectedDisconnect != nil {
		t.unexpectedDisconnect(t, err)
	}
}
