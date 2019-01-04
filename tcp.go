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
	ErrPunchFailed = errors.New("punching failed")
)

// PunchTCP tries to punch remote with specified reused local socket
func PunchTCP(timeout time.Duration, so net.Conn, remote string) (ret net.Conn, retErr error) {

	laddr := so.LocalAddr().String()
	l, err := reuse.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	start := time.Now()

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

	for {
		conn, err := reuse.DialWithTimeout("tcp", laddr, remote, time.Microsecond)
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
		if time.Now().After(start.Add(timeout)) {
			return
		}
		time.Sleep(time.Microsecond)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

}
