package readtest

/*
 Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/nsqio/go-nsq"
	//"github.com/nsqio/nsq/internal/app"
	//"github.com/nsqio/nsq/internal/version"
)

var (
	topic         = flag.String("topic", "", "NSQ topic")
	channel       = flag.String("channel", "", "NSQ channel")
	maxInFlight   = flag.Int("max-in-flight", 200, "max number of messages to allow in flight")
	totalMessages = flag.Int("n", 0, "total messages to show (will wait if starved)")
	printTopic    = flag.Bool("print-topic", false, "print topic name where message was received")

	nsqdTCPAddr    = flag.String("nsqd-tcp-address", "", "NSQ tcp address")
	lookupHTTPAddr = flag.String("lookup-http-address", "", "NSQ lookup http address")
)

type TailHandler struct {
	topicName     string
	totalMessages int
	messagesShown int
}

func (th *TailHandler) HandleMessage(m *nsq.Message) error {
	th.messagesShown++

	if *printTopic {
		_, err := os.Stdout.WriteString(th.topicName)
		if err != nil {
			log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
		}
		_, err = os.Stdout.WriteString(" | ")
		if err != nil {
			log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
		}
	}

	_, err := os.Stdout.Write(m.Body)
	if err != nil {
		log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
	}
	_, err = os.Stdout.WriteString("\n")
	if err != nil {
		log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
	}
	if th.totalMessages > 0 && th.messagesShown >= th.totalMessages {
		os.Exit(0)
	}
	return nil
}

func TestEventRead(t *testing.T) {
	cfg := nsq.NewConfig()

	flag.Var(&nsq.ConfigFlag{cfg}, "consumer-opt", "option to passthrough to nsq.Consumer (may be given multiple times, http://godoc.org/github.com/nsqio/go-nsq#Config)")
	flag.Parse()

	if *channel == "" {
		rand.Seed(time.Now().UnixNano())
		*channel = fmt.Sprintf("tail%06d#ephemeral", rand.Int()%999999)
	}

	if *nsqdTCPAddr == "" && *lookupHTTPAddr == "" {
		log.Fatal("--nsqd-tcp-address or --lookup-http-address required")
	}
	if *topic == "" {
		log.Fatal("--topic required")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Don't ask for more messages than we want
	if *totalMessages > 0 && *totalMessages < *maxInFlight {
		*maxInFlight = *totalMessages
	}

	//cfg.UserAgent = fmt.Sprintf("nsq_tail/%s go-nsq/%s", version.Binary, nsq.VERSION)
	cfg.MaxInFlight = *maxInFlight

	consumers := []*nsq.Consumer{}
	log.Printf("Adding consumer for topic: %s", *topic)

	consumer, err := nsq.NewConsumer(*topic, *channel, cfg)
	if err != nil {
		log.Fatal(err)
	}

	consumer.AddHandler(&TailHandler{topicName: *topic, totalMessages: *totalMessages})

	addrs := make([]string, 1)
	if *nsqdTCPAddr != "" {
		addrs[0] = *nsqdTCPAddr
		err = consumer.ConnectToNSQDs(addrs)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *lookupHTTPAddr != "" {
		addrs = make([]string, 1)
		addrs[0] = *lookupHTTPAddr
		err = consumer.ConnectToNSQLookupds(addrs)
		if err != nil {
			log.Fatal(err)
		}
	}

	consumers = append(consumers, consumer)

	<-sigChan

	for _, consumer := range consumers {
		consumer.Stop()
	}
	for _, consumer := range consumers {
		<-consumer.StopChan
	}
}
