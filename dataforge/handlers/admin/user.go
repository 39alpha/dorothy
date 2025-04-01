package admin

import (
	"fmt"
	"strconv"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UserForm struct {
	handlers.App
	Err error

	authUser *models.User
	userId   int
	user     *models.User
}

func (form *UserForm) Pre(c *fiber.Ctx) error {
	form.authUser, _ = c.Locals("AuthUser").(*models.User)
	if form.authUser == nil {
		return fiber.ErrUnauthorized
	} else if !form.authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}

	form.Err, _ = c.Locals("Error").(error)

	var err error
	form.userId, err = strconv.Atoi(c.Params("user"))
	if err != nil {
		return fiber.ErrNotFound
	}

	form.Err, _ = c.Locals("Error").(error)

	if form.user, err = form.DB().GetUserById(uint(form.userId)); err != nil {
		return handlers.GormToFiber(err)
	}

	return nil
}

func (form *UserForm) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverLogin(c, err)
}

func (form *UserForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/user", handlers.Bind(c, fiber.Map{
		"AuthUser": form.authUser,
		"User":     form.user,
		"Error":    form.Err,
	}), "layouts/main")
}

type UserUpdate struct {
	handlers.App

	update models.UpdateUserWithRole
	user   *models.User
}

func (page *UserUpdate) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	} else if !authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}

	var err error

	if err = c.BodyParser(&page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.user, err = page.DB().GetUserById(page.update.ID)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if page.user.RoleCode == models.AdminRole && page.update.Role != models.AdminRole {
		var admin_count int64
		if err := page.DB().Model(&models.User{}).Where("role_code = ?", models.AdminRole).Count(&admin_count).Error; err != nil {
			return handlers.GormToFiber(err)
		}

		if admin_count == 1 {
			return fmt.Errorf("%w: cannot remove the last admin's admin role", fiber.ErrBadRequest)
		}
	}

	return nil
}

func (page *UserUpdate) Run() error {
	if err := page.DB().UpdateUserWithRole(page.update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	return nil
}

func (page *UserUpdate) Post(c *fiber.Ctx) error {
	var err error
	page.user, err = page.DB().GetUserById(page.update.ID)
	return handlers.GormToFiber(err)
}

func (page *UserUpdate) Recover(c *fiber.Ctx, err error) error {
	return handlers.RecoverForm(&UserForm{
		App: page.App,
	}, c, err)
}

func (page *UserUpdate) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect(fmt.Sprintf("/admin/users/%d", page.user.ID))
}

func (page *UserUpdate) RenderJson(c *fiber.Ctx) error {
	return c.JSON(page.user)
}

func (page *UserUpdate) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}

type UserDelete struct {
	handlers.App

	user *models.User
}

func (page *UserDelete) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	} else if !authUser.HasAdminRole() {
		return fiber.ErrForbidden
	}

	userId, err := strconv.Atoi(c.Params("user"))
	if err != nil {
		return fiber.ErrNotFound
	}

	page.user, err = page.DB().GetUserById(uint(userId))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if page.user.RoleCode == models.AdminRole {
		var admin_count int64
		if err := page.DB().Model(&models.User{}).Where("role_code = ?", models.AdminRole).Count(&admin_count).Error; err != nil {
			return handlers.GormToFiber(err)
		}

		if admin_count == 1 {
			return fmt.Errorf("%w: cannot remove the last admin user", fiber.ErrBadRequest)
		}
	}

	return nil
}

func (page *UserDelete) Run() error {
	if err := page.DB().DeleteUser(page.user); err != nil {
		return fmt.Errorf(
			"%w: We couldn't delete the user for some reason. Try again later?",
			fiber.ErrInternalServerError,
		)
	}

	return nil
}

func (page *UserDelete) RenderHtml(c *fiber.Ctx) error {
	return c.Redirect("/admin/users")
}

func (page *UserDelete) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

func (page *UserDelete) RenderText(c *fiber.Ctx) error {
	return c.SendString("success")
}
