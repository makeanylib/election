package valkeyscript

import (
	"context"
	_ "embed"

	"github.com/valkey-io/valkey-go"
)

var (
	//go:embed delete_if_eq.lua
	deleteIfEqScriptContent string
	deleteIfEqScript        = valkey.NewLuaScript(deleteIfEqScriptContent)
)

func DeleteIfEq(
	ctx context.Context,
	client valkey.Client,
	key string,
	value string,
) error {
	resp := deleteIfEqScript.Exec(
		ctx,
		client,
		[]string{key},
		[]string{value},
	)
	if err := resp.Error(); err != nil {
		return err
	}
	return nil
}
