package events

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
	"github.com/nsqio/go-nsq"
	//"github.com/nsqio/nsq/internal/app"
	//"github.com/nsqio/nsq/internal/version"
	"fmt"
	log "github.com/sirupsen/logrus"
	"reflect"
)

// String returns the string form for a given LogLevel
func Publish(e EventInterface) error {
	cfg := nsq.NewConfig()
	if cfg == nil {
	}
	//cfg.UserAgent = fmt.Sprintf("to_nsq/%s go-nsq/%s", version.Binary, nsq.VERSION)
	cfg.UserAgent = fmt.Sprintf("go-nsq/%s", nsq.VERSION)

	log.Debugf("publishing %s message %s", reflect.TypeOf(e), e.String())
	log.Debugf("header %s ", e.GetHeader().String())

	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	log.Debug(string(b))

	var producer *nsq.Producer
	producer, err = nsq.NewProducer(e.GetHeader().SomeAddress, cfg)
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	err = producer.Publish(e.GetHeader().Topic[0], b)
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}

	return nil
}
