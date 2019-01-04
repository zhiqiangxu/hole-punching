package punching

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/zhiqiangxu/qrpc"

	reuse "github.com/libp2p/go-reuseport"
)

var (
	// ErrPunchFailed when punching failed
	ErrPunchFailed = errors.New("punching failed")
)

// PunchTCPByLocalPort tries to punch remote with specified local port
func PunchTCPByLocalPort(ctx context.Context, port uint16, remote string) (ret net.Conn, retErr error) {

	laddr := fmt.Sprintf("0.0.0.0:%d", port)
	l, err := reuse.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(ctx)
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
		conn, err := reuse.Dial("tcp", laddr, remote)
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
		time.Sleep(time.Microsecond)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

}
