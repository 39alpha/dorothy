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

	user *models.User
}

func (form *UserForm) Run(c *fiber.Ctx) error {
	if err := RequireAdmin(c); err != nil {
		return err
	}

	userId, err := strconv.Atoi(c.Params("user"))
	if err != nil {
		return fiber.ErrNotFound
	}

	if form.user, err = form.DB().GetUserById(uint(userId)); err != nil {
		return handlers.GormToFiber(err)
	}

	return nil
}

func (form *UserForm) RenderHtml(c *fiber.Ctx) error {
	return c.Render("admin/user", handlers.Bind(c, fiber.Map{
		"User": form.user,
	}), "layouts/main")
}

type UserUpdate struct {
	handlers.App

	user *models.User
}

func (page *UserUpdate) Run(c *fiber.Ctx) error {
	if err := RequireAdmin(c); err != nil {
		return err
	}

	var update models.UpdateUserWithRole
	if err := c.BodyParser(&update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	user, err := page.DB().GetUserById(update.ID)
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if user.RoleCode == models.AdminRole && update.Role != models.AdminRole {
		var admin_count int64
		if err := page.DB().Model(&models.User{}).Where("role_code = ?", models.AdminRole).Count(&admin_count).Error; err != nil {
			return handlers.GormToFiber(err)
		}

		if admin_count == 1 {
			return fmt.Errorf("%w: cannot remove the last admin's admin role", fiber.ErrBadRequest)
		}
	}

	if err := page.DB().UpdateUserWithRole(update); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	page.user, err = page.DB().GetUserById(update.ID)
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
}

func (page *UserDelete) Run(c *fiber.Ctx) error {
	if err := RequireAdmin(c); err != nil {
		return err
	}

	userId, err := strconv.Atoi(c.Params("user"))
	if err != nil {
		return fiber.ErrNotFound
	}

	user, err := page.DB().GetUserById(uint(userId))
	if err != nil {
		return handlers.GormToFiber(err)
	}

	if user.RoleCode == models.AdminRole {
		var admin_count int64
		if err := page.DB().Model(&models.User{}).Where("role_code = ?", models.AdminRole).Count(&admin_count).Error; err != nil {
			return handlers.GormToFiber(err)
		}

		if admin_count == 1 {
			return fmt.Errorf("%w: cannot remove the last admin user", fiber.ErrBadRequest)
		}
	}

	if err := page.DB().DeleteUser(user); err != nil {
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
