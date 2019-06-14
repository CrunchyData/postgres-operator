package events

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"reflect"
)

// String returns the string form for a given LogLevel
func Publish(e EventInterface) error {

	log.Debugf("publishing %s message %s", reflect.TypeOf(e), e.String())

	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		log.Errorf("Error: %s", err)
		return err
	}
	log.Debug(string(b))
	return nil
}
