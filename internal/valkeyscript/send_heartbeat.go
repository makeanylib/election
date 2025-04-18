package valkeyscript

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-go"
)

var (
	//go:embed send_heartbeat.lua
	sendHeartbeatScriptContent string
	sendHeartbeatScript        = valkey.NewLuaScript(sendHeartbeatScriptContent)
)

func SendHeartbeat(
	ctx context.Context,
	client valkey.Client,
	lockKey string,
	lockToken string,
	heartbeatTimeoutDuration time.Duration,
) error {
	resp := sendHeartbeatScript.Exec(
		ctx,
		client,
		[]string{lockKey},
		[]string{
			lockToken,
			strconv.FormatFloat(heartbeatTimeoutDuration.Seconds(), 'f', 0, 64),
		},
	)
	if err := resp.Error(); err != nil {
		return err
	}

	status, err := resp.AsInt64()
	if err != nil {
		return err
	}

	switch status {
	case 1:
		return nil
	case 2:
		return fmt.Errorf("forbidden")
	default:
		panic(fmt.Sprintf("unexpected status code: %d", status))
	}
}
