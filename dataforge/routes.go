package dataforge

import (
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/handlers/dataset"
	"github.com/39alpha/dorothy/dataforge/handlers/team"
)

type Method int

const (
	GET = iota
	POST
	DELETE
)

type Route struct {
	method   Method
	endpoint string
	handler  handlers.Handler
}

func Routes() []Route {
	return []Route{
		{GET, "/", &handlers.Home{}},

		{GET, "/register", &handlers.RegisterForm{}},
		{POST, "/register", &handlers.Register{}},
		{GET, "/login", &handlers.LoginForm{}},
		{POST, "/login", &handlers.Login{}},
		{GET, "/logout", &handlers.Logout{}},

		{GET, "/team/create", &team.CreateForm{}},
		{POST, "/team/create", &team.Create{}},
		{POST, "/team/availability", &team.Available{}},

		{GET, "/:team", &team.Team{}},
		{GET, "/:team/settings", &team.UpdateForm{}},
		{POST, "/:team/settings", &team.Update{}},
		{GET, "/:team/dataset/create", &dataset.CreateForm{}},
		{POST, "/:team/dataset/create", &dataset.Create{}},
		{POST, "/:team/dataset/availability", &dataset.Available{}},

		{GET, "/:team/:dataset", &dataset.Dataset{}},
		{POST, "/:team/:dataset/", &dataset.Receive{}},
		{DELETE, "/:team/:dataset/", &dataset.Delete{}},
		{GET, "/:team/:dataset/settings", &dataset.UpdateForm{}},
		{POST, "/:team/:dataset/settings", &dataset.Update{}},
	}
}
