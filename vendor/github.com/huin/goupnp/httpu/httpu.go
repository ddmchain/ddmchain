package httpu

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type HTTPUClient struct {
	connLock sync.Mutex 
	conn     net.PacketConn
}

func NewHTTPUClient() (*HTTPUClient, error) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}
	return &HTTPUClient{conn: conn}, nil
}

func NewHTTPUClientAddr(addr string) (*HTTPUClient, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.New("Invalid listening address")
	}
	conn, err := net.ListenPacket("udp", ip.String()+":0")
	if err != nil {
		return nil, err
	}
	return &HTTPUClient{conn: conn}, nil
}

func (httpu *HTTPUClient) Close() error {
	httpu.connLock.Lock()
	defer httpu.connLock.Unlock()
	return httpu.conn.Close()
}

func (httpu *HTTPUClient) Do(req *http.Request, timeout time.Duration, numSends int) ([]*http.Response, error) {
	httpu.connLock.Lock()
	defer httpu.connLock.Unlock()

	var requestBuf bytes.Buffer
	method := req.Method
	if method == "" {
		method = "GET"
	}
	if _, err := fmt.Fprintf(&requestBuf, "%s %s HTTP/1.1\r\n", method, req.URL.RequestURI()); err != nil {
		return nil, err
	}
	if err := req.Header.Write(&requestBuf); err != nil {
		return nil, err
	}
	if _, err := requestBuf.Write([]byte{'\r', '\n'}); err != nil {
		return nil, err
	}

	destAddr, err := net.ResolveUDPAddr("udp", req.Host)
	if err != nil {
		return nil, err
	}
	if err = httpu.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	for i := 0; i < numSends; i++ {
		if n, err := httpu.conn.WriteTo(requestBuf.Bytes(), destAddr); err != nil {
			return nil, err
		} else if n < len(requestBuf.Bytes()) {
			return nil, fmt.Errorf("httpu: wrote %d bytes rather than full %d in request",
				n, len(requestBuf.Bytes()))
		}
		time.Sleep(5 * time.Millisecond)
	}

	var responses []*http.Response
	responseBytes := make([]byte, 2048)
	for {

		n, _, err := httpu.conn.ReadFrom(responseBytes)
		if err != nil {
			if err, ok := err.(net.Error); ok {
				if err.Timeout() {
					break
				}
				if err.Temporary() {

					time.Sleep(10 * time.Millisecond)
					continue
				}
			}
			return nil, err
		}

		response, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(responseBytes[:n])), req)
		if err != nil {
			log.Print("httpu: error while parsing response: %v", err)
			continue
		}

		responses = append(responses, response)
	}
	return responses, err
}
