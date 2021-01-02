package cmd

/*
 Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type TailHandler struct {
	topicName     string
	totalMessages int
	messagesShown int
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Print watch information for the PostgreSQL Operator",
	Long: `WATCH allows you to watch event information for the postgres-operator. For example:
		pgo watch --pgo-event-address=localhost:14150  alltopic
		pgo watch alltopic`,
	Run: func(cmd *cobra.Command, args []string) {
		if Namespace == "" {
			Namespace = PGONamespace
		}

		log.Debug("watch called")
		watch(args)
	},
}

var PGOEventAddress string

func init() {
	RootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringVarP(&PGOEventAddress, "pgo-event-address", "a", "localhost:14150", "The address (host:port) where the event stream is.")
}

func watch(args []string) {
	log.Debugf("watch called %v", args)

	if len(args) == 0 {
		log.Fatal("topic is required")
	}

	topic := args[0]

	totalMessages := 0

	var channel string
	rand.Seed(time.Now().UnixNano())
	// #nosec: G404
	channel = fmt.Sprintf("tail%06d#ephemeral", rand.Int()%999999)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	cfg := nsq.NewConfig()
	cfg.MaxInFlight = 200

	consumers := []*nsq.Consumer{}
	log.Printf("Adding consumer for topic: %s", topic)

	consumer, err := nsq.NewConsumer(topic, channel, cfg)
	if err != nil {
		log.Fatal(err)
	}

	consumer.AddHandler(&TailHandler{topicName: topic, totalMessages: totalMessages})

	addrs := make([]string, 1)
	if PGOEventAddress != "" {
		addrs[0] = PGOEventAddress
		err = consumer.ConnectToNSQDs(addrs)
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

func (th *TailHandler) HandleMessage(m *nsq.Message) error {
	th.messagesShown++

	_, err := os.Stdout.WriteString(th.topicName)
	if err != nil {
		log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
	}
	_, err = os.Stdout.WriteString(" | ")
	if err != nil {
		log.Fatalf("ERROR: failed to write to os.Stdout - %s", err)
	}

	_, err = os.Stdout.Write(m.Body)
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
