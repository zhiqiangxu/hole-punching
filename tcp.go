package punching

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/zhiqiangxu/qrpc"

	reuse "github.com/zhiqiangxu/go-reuseport"
)

var (
	// ErrPunchFailed when punching failed
	ErrPunchFailed  = errors.New("punching failed")
	ErrInvalidParam = errors.New("invalid parameter")
)

// PunchTCP tries to punch remote with specified reused local socket
func PunchTCP(timeout int /*seconds*/, so net.Conn, remote string) (ret net.Conn, retErr error) {

	if timeout <= 0 {
		return nil, ErrInvalidParam
	}
	laddr := so.LocalAddr().String()
	l, err := reuse.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	var (
		lock sync.Mutex
		wg   sync.WaitGroup
	)

	defer func() {
		cancelFunc()
		wg.Wait()
		if ret == nil {
			retErr = ErrPunchFailed
		}
	}()

	qrpc.GoFunc(&wg, func() {
		for {
			l.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))
			conn, err := l.Accept()
			if err == nil {
				lock.Lock()
				if ret == nil {
					ret = conn
				} else {
					conn.Close()
				}
				lock.Unlock()
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	})

	start := time.Now()
	for {
		conn, err := reuse.DialWithTimeout("tcp", laddr, remote, time.Second)
		if err == nil {
			lock.Lock()
			if ret == nil {
				ret = conn
			} else {
				conn.Close()
			}
			lock.Unlock()
			return
		}

		fmt.Println("Dial err:", err)
		if time.Now().After(start.Add(time.Duration(timeout) * time.Second)) {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

}
