package ping

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/sagernet/sing-box/adapter"
)

// define parse function, helper function for ping
func parseFirstLine(buf []byte) (int, error) {
	bNext := buf
	var b []byte
	var err error
	for len(b) == 0 {
		if b, bNext, err = nextLine(bNext); err != nil {
			return 0, err
		}
	}

	// parse protocol
	n := bytes.IndexByte(b, ' ')
	if n < 0 {
		return 0, fmt.Errorf("cannot find whitespace in the first line of response %q", buf)
	}
	b = b[n+1:]

	// parse status code
	statusCode, n, err := parseUintBuf(b)
	if err != nil {
		return 0, fmt.Errorf("cannot parse response status code: %s. Response %q", err, buf)
	}
	if len(b) > n && b[n] != ' ' {
		return 0, fmt.Errorf("unexpected char at the end of status code. Response %q", buf)
	}

	if statusCode == http.StatusNoContent || statusCode == http.StatusOK {
		return len(buf) - len(bNext), nil
	}
	return 0, errors.New("wrong status code")
}

func nextLine(b []byte) ([]byte, []byte, error) {
	nNext := bytes.IndexByte(b, '\n')
	if nNext < 0 {
		return nil, nil, errors.New("need more data: cannot find trailing lf")
	}
	n := nNext
	if n > 0 && b[n-1] == '\r' {
		n--
	}
	return b[:n], b[nNext+1:], nil
}

func parseUintBuf(b []byte) (int, int, error) {
	n := len(b)
	if n == 0 {
		return -1, 0, errors.New("empty integer")
	}
	v := 0
	for i := 0; i < n; i++ {
		c := b[i]
		k := c - '0'
		if k > 9 {
			if i == 0 {
				return -1, i, errors.New("unexpected first char found. Expecting 0-9")
			}
			return v, i, nil
		}
		vNew := 10*v + int(k)
		// Test for overflow.
		if vNew < v {
			return -1, i, errors.New("too long int")
		}
		v = vNew
	}
	return v, n, nil
}

// CreateBlockNode 创建不可用节点
func CreateBlockNode(out adapter.Outbound) Node {
	return NewNodeBuilder(out).Build()
}

// CreateNodeWithPing 创建带延迟的节点
func CreateNodeWithPing(out adapter.Outbound, ping int64) Node {
	return NewNodeBuilder(out).
		WithPing(ping).
		Build()
}

// CreateNodeWithDetails 创建完整节点信息
func CreateNodeWithDetails(out adapter.Outbound, ping int64, ip, country string) Node {
	return NewNodeBuilder(out).
		WithPing(ping).
		WithIP(ip, country).
		Build()
}

// UpdateNodeRemoteIP 更新节点远程 IP 信息
func UpdateNodeRemoteIP(node *Node, ip, country string) Node {
	if node == nil {
		return Node{}
	}
	node.RemoteIP = ip
	node.RemoteCountry = country
	return *node
}
