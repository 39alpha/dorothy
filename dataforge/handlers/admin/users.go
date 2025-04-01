package admin

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type UserListing struct {
	handlers.App

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

func (page *UserListing) Pre(c *fiber.Ctx) error {
	if err := RequireAdmin(c); err != nil {
		return err
	}

	page.search = c.Query("search")

	pageNum := c.Query("page")
	if pageNum == "" {
		page.pageNum = 1
	} else {
		var err error
		page.pageNum, err = strconv.Atoi(pageNum)
		if err != nil {
			return fmt.Errorf("%w: invalid page number %q", fiber.ErrBadRequest, pageNum)
		}
	}

	perPage := c.Query("per_page")
	if perPage == "" {
		page.perPage = 20
	} else {
		var err error
		page.perPage, err = strconv.Atoi(perPage)
		if err != nil {
			return fmt.Errorf("%w: invalid per_page %q", fiber.ErrBadRequest, perPage)
		}
	}

	count_query := page.DB().Model(&models.User{})
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
		page.pageNum = 1
	}

	if handlers.Redirectable(c) {
		if pageNum != strconv.Itoa(page.pageNum) || perPage != strconv.Itoa(page.perPage) {
			return page.RedirectWithQueries(c, false, false)
		}
	}

	return nil
}

func (page *UserListing) Run() error {
	offset := page.perPage * (page.pageNum - 1)

	query := page.DB().
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
