package airshow

import (
	"fmt"
	"math/rand"
	"time"
)

type Airshow struct {
	workers []ImageWorker
	rand    *rand.Rand
}

type ImageWorker interface {
	GetImage() ([]byte, error)
	Run()
}

func New() *Airshow {
	a := &Airshow{}
	a.workers = make([]ImageWorker, 0)
	a.rand = rand.New(rand.NewSource(time.Now().Unix()))
	return a
}

func (a *Airshow) AddWorker(worker ImageWorker) {
	a.workers = append(a.workers, worker)
}

func (a *Airshow) getImageFromWorker() ([]byte, error) {
	index := a.rand.Int31n((int32)(len(a.workers)))
	return a.workers[index].GetImage()
}

func (a *Airshow) Run() {

	// bonjour ...
	addressChannel := make(chan string)
	go func() {
		err := searchBonjour("_airplay._tcp", addressChannel)
		panic("bonjour err", err)
	}()
	go func() {
		time.Sleep(time.Second * 5)
		close(addressChannel)
	}()

	addresses := make([]string, len(addressChannel))
	for address := range addressChannel {
		addresses = append(addresses, address)
	}

	for index, address := range addresses {
		fmt.Printf("%d) %s\n", index, address)
	}

	var answerIndex int
	_, err := fmt.Scanf("%d", &answerIndex)

	if err != nil {
		fmt.Println(err)
		return
	}

	address := addresses[0]

	conn, err := CreateConnection(address)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := conn.Handshake(); err != nil {
		fmt.Println(err)
		return
	}
	if err := conn.SubscribeSlideShow(); err != nil {
		fmt.Println(err)
		return
	}

	for _, worker := range a.workers {
		go worker.Run()
	}

	stop := make(chan bool)
	go func() {
		for {
			req, err := conn.ReadRequest()
			if err != nil {
				fmt.Println(err)
				return
			}
			// dispatch
			if req.URL.Path == "/slideshows/1/assets/1" {
				var data []byte
				var err error
				for {
					data, err = a.getImageFromWorker()
					if err == nil {
						break
					}
					fmt.Println("image roker err", err)
					time.Sleep(time.Second * 3)
				}
				conn.WriteSlideShowResponse(data)
			}
		}
	}()
	<-stop
}
