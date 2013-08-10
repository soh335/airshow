package airshow

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/kballard/go-osx-plist"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

type TestAirshowServer struct {
	listenOk     chan bool
	imageReceive chan []byte
	address      string
	errorReceive chan error
}

func NewTestAirshowServer(address string) *TestAirshowServer {
	s := &TestAirshowServer{}
	s.listenOk = make(chan bool)
	s.imageReceive = make(chan []byte)
	s.errorReceive = make(chan error)
	s.address = address
	return s
}

func (s *TestAirshowServer) start() {
	ln, err := net.Listen("tcp", s.address)

	if err != nil {
		// error
		fmt.Println(err)
		s.errorReceive <- err
		return
	}

	s.listenOk <- true

	for {
		con, err := ln.Accept()
		fmt.Println("accept...")
		if err != nil {
			// error
			fmt.Println(err)
			s.errorReceive <- err
			return
		}
		go s.readRequest(con)
	}
}

func TestSuite(t *testing.T) {
	address := net.JoinHostPort("127.0.0.1", "5050")
	server := NewTestAirshowServer(address)
	go func() {
		err := <-server.errorReceive
		fmt.Println("read err....", err)
		panic(err)
	}()
	go server.start()
	<-server.listenOk

	conn, err := CreateConnection(address)

	if err != nil {
		t.Error(err)
	}

	if err := conn.Handshake(); err != nil {
		t.Error(err)
	}

	if err := conn.SubscribeSlideShow(); err != nil {
		t.Error(err)
	}

	if err != nil {
		t.Error("temp file err", err)
	}

	image := createTempImage()

	go func() {
		fmt.Println("read...")
		req, err := conn.ReadSlideShowRequest()
		if err != nil {
			t.Error("read slide show request err", err)
		}
		fmt.Println("req", req, "writing image........")
		conn.WriteSlideShowResponse(image)
	}()
	<-server.imageReceive
	conn.con.Close()
}

func createTempImage() []byte {
	b := bytes.NewBuffer(make([]byte, 0))
	m := image.NewRGBA(image.Rect(0, 0, 100, 100))
	blue := color.RGBA{0, 0, 255, 255}
	draw.Draw(m, m.Bounds(), &image.Uniform{blue}, image.ZP, draw.Src)
	jpeg.Encode(b, m, nil)

	return b.Bytes()
}

func (s *TestAirshowServer) readRequest(c io.ReadWriteCloser) {
	for {
		br := bufio.NewReader(c)
		fmt.Println("test server read request")
		req, err := http.ReadRequest(br)

		if err != nil {
			fmt.Println("read request err", req, err)
			s.errorReceive <- err
			return
		}

		defer req.Body.Close()

		if req.Method == "POST" && req.URL.Path == "/reverse" {
			if err := s.responseHandshake(c, req); err != nil {
				fmt.Println("post reverse err", err)
				s.errorReceive <- err
				return
			}
		}

		if req.Method == "PUT" && req.URL.Path == "/slideshows/1" {
			if err := s.responseSlideshows(c, req); err != nil {
				fmt.Println("put slideshows err", err)
				s.errorReceive <- err
				return
			}
			break
		}
	}

	time.Sleep(time.Second * 1)

	for {
		if err := s.requestSlideshowAsset(c); err != nil {
			fmt.Println(err)
			s.errorReceive <- err
			return
		}
	}

	c.Close()
}

func responseHandshakeTest(req *http.Request) error {
	spec := map[string]string{
		"Upgrade":         "PTTH/1.0",
		"X-Apple-Purpose": "slideshow",
		"Content-Length":  "0",
		"Connection":      "Upgrade",
	}
	for key, value := range spec {
		if req.Header.Get(key) != value {
			return errors.New(fmt.Sprintf("should %s == %s but %s", key, value, req.Header.Get(key)))
		}
	}

	if req.Header.Get("X-Apple-Session-ID") == "" {
		return errors.New("should X-Apple-Session-ID header require")
	}

	return nil
}

func (s *TestAirshowServer) responseHandshake(c io.ReadWriteCloser, req *http.Request) error {
	defer req.Body.Close()

	if err := responseHandshakeTest(req); err != nil {
		return err
	}

	bw := bufio.NewWriter(c)

	bw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bw.WriteString("Upgrade: PTTH/1.0\r\n")
	bw.WriteString("Connection: Upgrade\r\n")
	bw.WriteString("\r\n")

	if err := bw.Flush(); err != nil {
		return err
	}

	fmt.Println("<- http/1.1 101 switching protocols")

	return nil
}

func responseSlideshowsTest(req *http.Request) error {
	bdata, err := ioutil.ReadAll(req.Body)

	if err != nil {
		return err
	}

	spec := map[string]string{
		"Content-Type":   "text/x-apple-plist+xml",
		"Content-Length": strconv.Itoa(len(bdata)),
	}

	for key, value := range spec {
		if req.Header.Get(key) != value {
			return errors.New(fmt.Sprintf("should %s == %s but %s", key, value, req.Header.Get(key)))
		}
	}

	var result interface{}
	format, err := plist.Unmarshal(bdata, &result)

	if err != nil {
		return err
	}

	if format != plist.XMLFormat {
		return errors.New("should format is xml")
	}

	if result.(map[string]interface{})["state"].(string) != "playing" {
		return errors.New("state should be playing")
	}

	if result.(map[string]interface{})["settings"] == nil {
		return errors.New("should has settings key")
	}

	return nil
}

func (s *TestAirshowServer) responseSlideshows(c io.ReadWriteCloser, req *http.Request) error {
	defer req.Body.Close()

	if err := responseSlideshowsTest(req); err != nil {
		return err
	}

	bw := bufio.NewWriter(c)

	bdata, _ := plist.Marshal(make(map[interface{}]interface{}), plist.XMLFormat)

	bw.WriteString("HTTP/1.1 200 OK\r\n")
	bw.WriteString("Content-Type: text/x-apple-plist+xml\r\n")
	fmt.Fprintf(bw, "Content-Length: %d\r\n", len(bdata))
	bw.WriteString("\r\n")

	if err := bw.Flush(); err != nil {
		return err
	}

	if len(bdata) > 0 {
		bw.Write(bdata)
		if err := bw.Flush(); err != nil {
			return err
		}
	}
	fmt.Println("<-res http/1.1 200 ok")

	return nil
}

func (s *TestAirshowServer) requestSlideshowAsset(c io.ReadWriteCloser) error {
	bw := bufio.NewWriter(c)
	br := bufio.NewReader(c)

	bw.WriteString("GET /slideshows/1/assets/1 HTTP/1.1\r\n")
	bw.WriteString("Content-Length: 0\r\n")
	bw.WriteString("Accept: application/x-apple-binary-plist\r\n")
	bw.WriteString("X-Apple-Session-ID: hogehoge\r\n")
	bw.WriteString("\r\n")

	if err := bw.Flush(); err != nil {
		return err
	}

	fmt.Println("->get slideshow/1/assets/1")
	resp, err := http.ReadResponse(br, &http.Request{Method: "GET"})
	fmt.Println("slideshow/1/assets", resp, err)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("invalid status")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	var result interface{}

	format, err := plist.Unmarshal(body, &result)

	if err != nil {
		return err
	}

	if format != plist.BinaryFormat {
		return errors.New("should binary format")
	}

	info := result.(map[string]interface{})["info"].(map[string]interface{})
	if !(info["key"].(int64) == 1 && info["id"].(int64) == 1) {
		return errors.New("should info.key and info.id is 1")
	}

	data := result.(map[string]interface{})["data"].([]byte)
	_, err = jpeg.Decode(bytes.NewBuffer(data))

	if err != nil {
		return err
	}

	s.imageReceive <- body

	return nil
}
