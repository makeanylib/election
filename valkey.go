package election

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/makeanylib/malgo/maybe"
	"github.com/valkey-io/valkey-go"
	"gopkg.makeany.app/election/internal/valkeyscript"
)

type ValkeyElector struct {
	client                   valkey.Client
	prefix                   maybe.T[string]
	name                     string
	heartbeatDuration        time.Duration
	heartbeatTimeoutDuration time.Duration
}

var _ Elector = (*ValkeyElector)(nil)

func NewValkeyElector(
	client valkey.Client,
	prefix maybe.T[string],
	name string,
	heartbeatDuration time.Duration,
) *ValkeyElector {
	return &ValkeyElector{
		client:                   client,
		prefix:                   prefix,
		name:                     name,
		heartbeatDuration:        heartbeatDuration,
		heartbeatTimeoutDuration: heartbeatDuration * 2,
	}
}

func (v *ValkeyElector) Elect(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	errch := make(chan error, 1)
	notifych := make(chan struct{}, 1)
	go func() {
		defer close(errch)

		err := v.client.Receive(
			ctx,
			v.client.B().
				Subscribe().
				// TODO: support prefix
				Channel(v.name).
				Build(),
			func(msg valkey.PubSubMessage) {
				fmt.Println("Received message:", msg)
				select {
				case notifych <- struct{}{}:
					// DO NOTHING.
				case <-ctx.Done():
					return
				}
			},
		)
		if err != nil {
			errch <- err
		}
		errch <- nil
	}()

	for {
		token := uuid.NewString()
		elected := true
		resp := v.client.Do(
			ctx,
			v.client.B().
				Set().
				// TODO: support prefix
				Key(v.name).
				Value(token).
				Nx().
				Ex(v.heartbeatTimeoutDuration).
				Build(),
		)
		if err := resp.Error(); err != nil {
			if valkey.IsValkeyNil(err) {
				elected = false
			} else {
				return err
			}
		}

		if elected {
			err := func() error {
				defer func() {
					resp := v.client.Do(
						ctx,
						v.client.B().
							Publish().
							Channel(v.name).
							Message(":P").
							Build(),
					)
					if err := resp.Error(); err != nil {
						// TODO: logging error
						fmt.Println("error while publishing message:", err)
					}
				}()

				defer func() {
					err := valkeyscript.DeleteIfEq(
						ctx,
						v.client,
						v.name,
						token,
					)
					if err != nil {
						// TODO: logging error
						fmt.Println("error while deleting token:", err)
					}
				}()

				callbackCtx, cancelCallbackCtx := context.WithCancel(ctx)
				go func() {
					defer cancelCallbackCtx()

					for {
						fmt.Println("Sending heartbeat")
						err := valkeyscript.SendHeartbeat(
							callbackCtx,
							v.client,
							v.name,
							token,
							v.heartbeatTimeoutDuration,
						)
						if err != nil {
							// TODO: logging error
							fmt.Println(err)
							return
						}

						select {
						case <-callbackCtx.Done():
							return
						case <-time.After(v.heartbeatDuration):
							// DO NOTHING.
						}
					}
				}()
				err := fn(callbackCtx)
				cancelCallbackCtx()
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errch:
				return err
			default:
				continue
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errch:
			return err
		case <-notifych:
			continue
		case <-time.After(5 * time.Second /* TODO: make it configurable */):
			continue
		}
	}
}
