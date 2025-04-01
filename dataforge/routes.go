package dataforge

import (
	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/handlers/admin"
	"github.com/39alpha/dorothy/dataforge/handlers/dataset"
	"github.com/39alpha/dorothy/dataforge/handlers/profile"
	"github.com/39alpha/dorothy/dataforge/handlers/team"
	"github.com/39alpha/dorothy/dataforge/handlers/user"
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
		{GET, "/reset-password", &handlers.ResetPasswordForm{}},
		{POST, "/reset-password", &handlers.ResetPassword{}},

		{GET, "/admin", &admin.Dashboard{}},
		{GET, "/admin/users", &admin.UserListing{}},
		{GET, "/admin/users/create", &admin.UserCreateForm{}},
		{POST, "/admin/users/create", &admin.UserCreate{}},
		{GET, "/admin/users/:user", &admin.UserForm{}},
		{POST, "/admin/users/:user", &admin.UserUpdate{}},
		{DELETE, "/admin/users/:user", &admin.UserDelete{}},

		{GET, "/profile", &profile.Form{}},
		{POST, "/profile", &profile.Update{}},
		{POST, "/profile/change-password", &profile.ChangePassword{}},

		{GET, "/user/search", &user.Search{}},

		{GET, "/team/create", &team.CreateForm{}},
		{POST, "/team/create", &team.Create{}},
		{POST, "/team/availability", &team.Available{}},

		{GET, "/:team", &team.Team{}},
		{DELETE, "/:team", &team.Delete{}},
		{GET, "/:team/settings", &team.UpdateForm{}},
		{POST, "/:team/settings", &team.Update{}},
		{POST, "/:team/privilege", &team.CreatePrivilege{}},
		{DELETE, "/:team/privilege", &team.DeletePrivilege{}},
		{GET, "/:team/dataset/create", &dataset.CreateForm{}},
		{POST, "/:team/dataset/create", &dataset.Create{}},
		{POST, "/:team/dataset/availability", &dataset.Available{}},

		{GET, "/:team/:dataset", &dataset.Dataset{}},
		{POST, "/:team/:dataset", &dataset.Receive{}},
		{DELETE, "/:team/:dataset", &dataset.Delete{}},
		{GET, "/:team/:dataset/settings", &dataset.UpdateForm{}},
		{POST, "/:team/:dataset/settings", &dataset.Update{}},
		{POST, "/:team/:dataset/privilege", &dataset.CreatePrivilege{}},
		{DELETE, "/:team/:dataset/privilege", &dataset.DeletePrivilege{}},
	}
}
