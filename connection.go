package airshow

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/kballard/go-osx-plist"
	"github.com/nu7hatch/gouuid"
	"io"
	"net"
	"net/http"
)

type Connection struct {
	con       io.ReadWriteCloser
	sessionId string
}

func CreateConnection(address string) (*Connection, error) {
	con, err := net.Dial("tcp", address)

	if err != nil {
		return nil, err
	}

	u4, _ := uuid.NewV4()

	return &Connection{con, u4.String()}, nil
}

func (c *Connection) Handshake() error {
	bw := bufio.NewWriter(c.con)
	br := bufio.NewReader(c.con)

	bw.WriteString("POST /reverse HTTP/1.1\r\n")
	bw.WriteString("Upgrade: PTTH/1.0\r\n")
	bw.WriteString("X-Apple-Purpose: slideshow\r\n")
	bw.WriteString("Content-Length: 0\r\n")
	fmt.Fprintf(bw, "X-Apple-Session-ID: %s\r\n", c.sessionId)
	bw.WriteString("Connection: Upgrade\r\n")
	bw.WriteString("\r\n")

	if err := bw.Flush(); err != nil {
		return err
	}

	resp, err := http.ReadResponse(br, &http.Request{Method: "POST"})

	if err != nil {
		return err
	}

	if resp.Header.Get("Connection") != "Upgrade" {
		return errors.New("handshake error")
	}
	return nil
}

func (c *Connection) SubscribeSlideShow() error {
	bw := bufio.NewWriter(c.con)
	br := bufio.NewReader(c.con)

	data := map[string]interface{}{
		"state": "playing",
		"settings": map[string]interface{}{
			"slideDuration": 3,
			"theme":         "Origami",
		},
	}
	bdata, err := plist.Marshal(data, plist.XMLFormat)

	if err != nil {
		return err
	}

	bw.WriteString("PUT /slideshows/1 HTTP/1.1\r\n")
	bw.WriteString("Content-Type: text/x-apple-plist+xml\r\n")
	fmt.Fprintf(bw, "X-Apple-Session-ID: %s\r\n", c.sessionId)
	fmt.Fprintf(bw, "Content-Length: %d\r\n", len(bdata))
	bw.WriteString("\r\n")

	if err := bw.Flush(); err != nil {
		return err
	}

	bw.Write(bdata)

	if err := bw.Flush(); err != nil {
		return err
	}

	resp, err := http.ReadResponse(br, &http.Request{Method: "POST"})
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) ReadRequest() (*http.Request, error) {
	br := bufio.NewReader(c.con)
	return http.ReadRequest(br)
}

func (c *Connection) WriteSlideShowResponse(image []byte) error {
	data := map[string]interface{}{
		"data": image,
		"info": map[string]interface{}{
			"key": 1,
			"id":  1,
		},
	}
	bdata, err := plist.Marshal(data, plist.BinaryFormat)

	if err != nil {
		return err
	}

	bw := bufio.NewWriter(c.con)

	bw.WriteString("HTTP/1.1 200 OK\r\n")
	bw.WriteString("Content-Type: application/x-apple-binary-plist\r\n")
	fmt.Fprintf(bw, "Content-Length: %d\r\n", len(bdata))
	bw.WriteString("\r\n")

	if err := bw.Flush(); err != nil {
		return err
	}

	bw.Write(bdata)

	if err := bw.Flush(); err != nil {
		return err
	}

	return nil
}
