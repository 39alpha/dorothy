package core

import (
	"encoding/json"

	"github.com/kataras/iris/v12"
)

func RecordBody(ctx iris.Context) {
	ctx.RecordRequestBody(true)
	ctx.Next()
}

func WithConfig(config *Config) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Values().Set("config", config)
		ctx.Next()
	}
}

func GetConfig(ctx iris.Context) *Config {
	return ctx.Values().Get("config").(*Config)
}

func WithDbSession(session *DatabaseSession) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Values().Set("db_session", session)
		ctx.Next()
	}
}

func GetDbSession(ctx iris.Context) *DatabaseSession {
	return ctx.Values().Get("db_session").(*DatabaseSession)
}

func ParseBody(ctx iris.Context) {
	if ctx.Method() != "GET" {
		body, err := ctx.GetBody()
		if err != nil {
			ctx.StopWithError(iris.StatusBadRequest, err)
			return
		}

		var parsed iris.Map
		if err := json.Unmarshal(body, &parsed); err != nil {
			ctx.StopWithError(iris.StatusBadRequest, err)
			return
		}

		ctx.Values().Set("jsonbody", parsed)
	}
	ctx.Next()
}
