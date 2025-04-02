package admin

import (
	"errors"
	"fmt"
	"math"
	"net/url"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UserListing struct {
	users    []models.User
	search   string
	pageNum  int
	perPage  int
	numPages int
}

func (page *UserListing) PathWithQueries(c *fiber.Ctx, useQueries, escape bool) string {
	path := fmt.Sprintf("%s?%s", c.Path(), string(c.Context().QueryArgs().QueryString()))
	if !useQueries {
		path = fmt.Sprintf("%s?search=%s&page=%d&per_page=%d", c.Path(), page.search, page.pageNum, page.perPage)
	}

	if escape {
		return url.QueryEscape(path)
	}
	return path
}

func (page *UserListing) RedirectWithQueries(c *fiber.Ctx, useQueries, escape bool) error {
	return c.Redirect(page.PathWithQueries(c, useQueries, escape))
}

func (page *UserListing) Run(c *fiber.Ctx) error {
	db := handlers.GetDB(c)

	if err := RequireAdmin(c); err != nil {
		return err
	}

	page.search = c.Query("search")

	redirect := false
	page.pageNum = c.QueryInt("page", 1)
	if page.pageNum < 1 {
		redirect = true
		page.pageNum = 1
	}

	page.perPage = c.QueryInt("per_page", 20)
	if page.perPage < 1 {
		redirect = true
		page.perPage = 20
	}

	count_query := db.Model(&models.User{})
	if page.search != "" {
		pattern := fmt.Sprintf("%%%s%%", page.search)
		count_query = count_query.
			Where("name LIKE ? OR email LIKE ? OR orcid LIKE ?", pattern, pattern, pattern)
	}

	var user_count int64
	if err := count_query.Count(&user_count).Error; err != nil {
		return fmt.Errorf("%w: failed to get users", handlers.GormToFiber(err))
	}

	page.numPages = int(math.Ceil(float64(user_count) / float64(page.perPage)))
	if page.pageNum > page.numPages {
		redirect = true
		page.pageNum = 1
	}

	if handlers.Redirectable(c) && redirect {
		return page.RedirectWithQueries(c, false, false)
	}

	offset := page.perPage * (page.pageNum - 1)

	query := db.
		Omit("PasswordHash").
		Order("name").
		Offset(offset).
		Limit(page.perPage)

	if page.search != "" {
		pattern := fmt.Sprintf("%%%s%%", page.search)
		query = query.
			Where("name LIKE ? OR email LIKE ? OR orcid LIKE ?", pattern, pattern, pattern)
	}

	if err := query.Find(&page.users).Error; err != nil {
		return fmt.Errorf("%w: failed to get users", handlers.GormToFiber(err))
	}

	return nil
}

func (page *UserListing) Recover(c *fiber.Ctx, err error) error {
	var e *fiber.Error
	if errors.As(err, &e) && e == fiber.ErrUnauthorized && handlers.Redirectable(c) {
		return c.Status(e.Code).Redirect("/login?Redirect=" + page.PathWithQueries(c, true, true))
	}

	return err
}

func (page *UserListing) RenderHtml(c *fiber.Ctx) error {
	params := handlers.Bind(c, fiber.Map{
		"Users":    page.users,
		"Search":   page.search,
		"Page":     page.pageNum,
		"PerPage":  page.perPage,
		"NumPages": page.numPages,
	})
	if page.pageNum > 1 {
		params["PrevPage"] = page.pageNum - 1
	}
	if page.pageNum < page.numPages {
		params["NextPage"] = page.pageNum + 1
	}

	return c.Render("admin/users", params, "layouts/main")
}

func (page *UserListing) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"users":     page.users,
		"search":    page.search,
		"page":      page.pageNum,
		"per_page":  page.perPage,
		"num_pages": page.numPages,
	})
}
