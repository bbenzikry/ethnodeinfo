package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "wss://stats.goerli.net/primus/", "ethstats address")

type NodeDetails struct {
	ID      string `json:"id"`
	Trusted bool   `json:"trusted"`
	Info    struct {
		Name             string `json:"name"`
		Node             string `json:"node"`
		Port             int    `json:"port"`
		Net              string `json:"net"`
		Protocol         string `json:"protocol"`
		API              string `json:"api"`
		Os               string `json:"os"`
		OsV              string `json:"os_v"`
		Client           string `json:"client"`
		CanUpdateHistory bool   `json:"canUpdateHistory"`
		Data             struct {
		} `json:"data"`
		IP string `json:"ip"`
	} `json:"info"`
	Geo struct {
		Range   []int     `json:"range"`
		Country string    `json:"country"`
		Region  string    `json:"region"`
		City    string    `json:"city"`
		Ll      []float64 `json:"ll"`
		Metro   int       `json:"metro"`
	} `json:"geo"`
	Stats struct {
		Active   bool  `json:"active"`
		Mining   bool  `json:"mining"`
		Hashrate int   `json:"hashrate"`
		Peers    int   `json:"peers"`
		Pending  int   `json:"pending"`
		GasPrice int64 `json:"gasPrice"`
		// Block    struct {
		// 	Number          int    `json:"number"`
		// 	Hash            string `json:"hash"`
		// 	ParentHash      string `json:"parentHash"`
		// 	Timestamp       int    `json:"timestamp"`
		// 	Miner           string `json:"miner"`
		// 	GasUsed         int    `json:"gasUsed"`
		// 	GasLimit        int    `json:"gasLimit"`
		// 	Difficulty      string `json:"difficulty"`
		// 	TotalDifficulty string `json:"totalDifficulty"`
		// 	Transactions    []struct {
		// 		Hash string `json:"hash"`
		// 	} `json:"transactions"`
		// 	TransactionsRoot string        `json:"transactionsRoot"`
		// 	StateRoot        string        `json:"stateRoot"`
		// 	Uncles           []interface{} `json:"uncles"`
		// 	Trusted          bool          `json:"trusted"`
		// 	Arrived          int64         `json:"arrived"`
		// 	Received         int64         `json:"received"`
		// 	Propagation      int           `json:"propagation"`
		// 	Fork             int           `json:"fork"`
		// } `json:"block"`
		Syncing        bool   `json:"syncing"`
		PropagationAvg int    `json:"propagationAvg"`
		Latency        string `json:"latency"`
		Uptime         int    `json:"uptime"`
	} `json:"stats"`
	History []int `json:"history"`
	Uptime  struct {
		Started    int64 `json:"started"`
		Up         int   `json:"up"`
		Down       int   `json:"down"`
		LastStatus bool  `json:"lastStatus"`
		LastUpdate int64 `json:"lastUpdate"`
	} `json:"uptime"`
	Spark string `json:"spark"`
}

type EthStatsMsg struct {
	Emit []interface{} `json:"emit"`
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	println("Connecting to", *addr)
	u, err := url.Parse(*addr)
	if err != nil {
		log.Fatal("Cannot parse URL")
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				return
			}
			msg := &EthStatsMsg{}
			json.Unmarshal(message, &msg)
			if msg.Emit != nil {
				if msg.Emit[0] == "init" {
					nodes := msg.Emit[1].(map[string]interface{})["nodes"]
					marshalled, err := json.Marshal(nodes)
					if err != nil {
						log.Fatal("cannot marshal node list", err)
					}
					nodeList := make([]*NodeDetails, 0)
					err = json.Unmarshal(marshalled, &nodeList)
					if err != nil {
						log.Fatal("cannot unmarshal node list", err)
					}
					marshalled, err = json.MarshalIndent(nodeList, "", "  ")
					if err != nil {
						log.Fatal("cannot marshal indent node list", err)
					}
					fmt.Print(string(marshalled))
					done <- struct{}{}
				}
			}
		}
	}()

	err = c.WriteMessage(websocket.TextMessage, []byte("{\"emit\":[\"ready\"]}"))
	if err != nil {
		log.Fatal("cannot write message to transport", err)
	}

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
