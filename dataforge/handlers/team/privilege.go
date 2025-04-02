package team

import (
	"fmt"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type DeletePrivilege struct {
	handlers.App
}

func (page *DeletePrivilege) Run(c *fiber.Ctx) error {
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	var payload struct {
		Id     uint
		UserId uint
	}
	if err := c.BodyParser(&payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	privilege := models.UserTeamPrivilege{
		UserID: payload.UserId,
		TeamID: payload.Id,
	}

	team, err := page.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	} else if team.ID != privilege.TeamID {
		return fiber.ErrBadRequest
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	users, err := page.DB().GetUsersWithTeamAccess(*team)
	if err != nil {
		return fmt.Errorf(
			"%w: cannot safely perform the request. Please try again later.",
			fiber.ErrInternalServerError,
		)
	}

	adminUsers := []models.User{}
	isAdmin := false
	var thisUser *models.User
	for _, user := range users {
		if user.ID == privilege.UserID {
			thisUser = &user
		}
		privileges := user.TeamPrivileges
		if len(privileges) == 1 && privileges[0].Privilege.Code == models.AdminPrivilege {
			isAdmin = thisUser == &user
			adminUsers = append(adminUsers, user)
		}
	}

	if len(users) == 1 {
		return fmt.Errorf("%w: cannot remove the last user's privileges", fiber.ErrBadRequest)
	} else if thisUser == nil {
		return fmt.Errorf("%w: we cannot find that user's privilege", fiber.ErrNotFound)
	} else if isAdmin && len(adminUsers) == 1 {
		return fmt.Errorf("%w: cannot remove the last admin user's privileges", fiber.ErrBadRequest)
	}

	if err := page.DB().DeleteTeamPrivilege(privilege); err != nil {
		return fmt.Errorf("%w: could not remove the privilege", handlers.GormToFiber(err))
	}

	return nil
}

func (page *DeletePrivilege) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

type CreatePrivilege struct {
	handlers.App
}

func (page *CreatePrivilege) Run(c *fiber.Ctx) error {
	authUser := handlers.GetAuthUser(c)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	var payload struct {
		Id            uint
		UserId        uint
		PrivilegeCode models.PrivilegeCode
	}
	if err := c.BodyParser(&payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	privilege := models.UserTeamPrivilege{
		UserID:        payload.UserId,
		TeamID:        payload.Id,
		PrivilegeCode: payload.PrivilegeCode,
	}

	team, err := page.DB().GetTeam(authUser, c.Params("team"))
	if err != nil {
		return handlers.GormToFiber(err)
	} else if team.ID != privilege.TeamID {
		return fiber.ErrBadRequest
	}

	if !authUser.CanManageTeam(*team) {
		return fiber.ErrForbidden
	}

	users, err := page.DB().GetUsersWithTeamAccess(*team)
	if err != nil {
		return fmt.Errorf(
			"%w: cannot safely perform the request. Please try again later.",
			fiber.ErrInternalServerError,
		)
	}

	adminUsers := []models.User{}
	isAdmin := false
	var thisUser *models.User
	for _, user := range users {
		if user.ID == privilege.UserID {
			thisUser = &user
		}
		privileges := user.TeamPrivileges
		if len(privileges) == 1 && privileges[0].Privilege.Code == models.AdminPrivilege {
			isAdmin = thisUser == &user
			adminUsers = append(adminUsers, user)
		}
	}

	if thisUser == nil {
		return fmt.Errorf("%w: we cannot find that user's privilege", fiber.ErrNotFound)
	} else if isAdmin && len(adminUsers) == 1 && thisUser.TeamPrivileges[0].Privilege.Code != privilege.PrivilegeCode {
		return fmt.Errorf("%w: cannot remove the last admin user's admin privileges", fiber.ErrBadRequest)
	}

	if err := page.DB().UpdateTeamPrivilege(privilege); err != nil {
		return fmt.Errorf("%w: could not update the privilege", handlers.GormToFiber(err))
	}

	return nil
}

func (page *CreatePrivilege) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}
