package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/foliagecp/easyjson"
	sf "github.com/foliagecp/sdk/statefun"
	sfp "github.com/foliagecp/sdk/statefun/plugins"
	"github.com/nats-io/nats.go"
)

type OpError struct {
	StatusCode int
	Details    string
}

func (oe *OpError) Error() string {
	return fmt.Sprintf("%d: %s", oe.StatusCode, oe.Details)
}

func buildNatsData(callerTypename string, callerID string, payload *easyjson.JSON, options *easyjson.JSON) []byte {
	data := easyjson.NewJSONObject()
	data.SetByPath("caller_typename", easyjson.NewJSON(callerTypename))
	data.SetByPath("caller_id", easyjson.NewJSON(callerID))
	if payload != nil {
		data.SetByPath("payload", *payload)
	}
	if options != nil {
		data.SetByPath("options", *options)
	}
	return data.ToBytes()
}

func getRequestFunc(nc *nats.Conn, NatsRequestTimeoutSec int) sfp.SFRequestFunc {
	return func(r sfp.RequestProvider, targetTypename string, targetID string, payload *easyjson.JSON, options *easyjson.JSON) (*easyjson.JSON, error) {
		targetDomain := ""
		tokens := strings.Split(targetID, sf.ObjectIDDomainSeparator)
		if len(tokens) == 2 {
			targetDomain = tokens[0]
		}

		resp, err := nc.Request(
			fmt.Sprintf("request.%s.%s.%s", targetDomain, targetTypename, targetID),
			buildNatsData("cli", "cli", payload, options),
			time.Duration(NatsRequestTimeoutSec)*time.Second,
		)
		if err == nil {
			if j, ok := easyjson.JSONFromBytes(resp.Data); ok {
				return &j, err
			}
			return nil, fmt.Errorf("response from function typename \"%s\" with id \"%s\" is not a json", targetTypename, targetID)
		}
		return nil, err
	}
}
