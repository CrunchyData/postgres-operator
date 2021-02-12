package events

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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/nsqio/go-nsq"
	log "github.com/sirupsen/logrus"
)

// String returns the string form for a given LogLevel
func Publish(e EventInterface) error {
	eventAddr := os.Getenv("EVENT_ADDR")
	if eventAddr == "" {
		return errors.New("EVENT_ADDR not set")
	}
	if os.Getenv("DISABLE_EVENTING") == "true" {
		log.Debugf("eventing disabled")
		return nil
	}

	cfg := nsq.NewConfig()
	// cfg.UserAgent = fmt.Sprintf("to_nsq/%s go-nsq/%s", version.Binary, nsq.VERSION)
	cfg.UserAgent = fmt.Sprintf("go-nsq/%s", nsq.VERSION)

	log.Debugf("publishing %s message %s", reflect.TypeOf(e), e.String())
	log.Debugf("header %s ", e.GetHeader().String())

	header := e.GetHeader()
	header.Timestamp = time.Now()

	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	log.Debug(string(b))

	var producer *nsq.Producer
	producer, err = nsq.NewProducer(eventAddr, cfg)
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}

	topics := e.GetHeader().Topic
	if len(topics) == 0 {
		log.Errorf("Error: topics list is empty and is required to publish")
		return err
	}

	for i := 0; i < len(topics); i++ {
		err = producer.Publish(topics[i], b)
		if err != nil {
			log.Errorf("Error: %s", err)
			return err
		}
	}

	// always publish to the All topic
	err = producer.Publish(EventTopicAll, b)
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}

	return nil
}
