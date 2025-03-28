package dataset

import (
	"fmt"
	"strconv"

	"github.com/39alpha/dorothy/dataforge/handlers"
	"github.com/39alpha/dorothy/dataforge/models"
	"github.com/gofiber/fiber/v2"
)

type DeletePrivilege struct {
	handlers.ErrorHandler

	payload struct {
		Id     string
		UserId string
	}

	datasetId uint
	userId    uint
}

func (page *DeletePrivilege) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	if err := c.BodyParser(&page.payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if datasetId, err := strconv.Atoi(page.payload.Id); err != nil {
		return fmt.Errorf(
			"%w: the provided id (%q) is not a positive integer",
			fiber.ErrBadRequest,
			page.payload.Id,
		)
	} else {
		page.datasetId = uint(datasetId)
	}

	if userId, err := strconv.Atoi(page.payload.UserId); err != nil {
		return fmt.Errorf(
			"%w: the provided userId (%q) is not a positive integer",
			fiber.ErrBadRequest,
			page.payload.UserId,
		)
	} else {
		page.userId = uint(userId)
	}

	dataset, err := page.DB().GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	} else if dataset.ID != page.datasetId {
		return fiber.ErrBadRequest
	}

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	users, err := page.DB().GetUsersWithDatasetAccess(*dataset)
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
		if user.ID == page.userId {
			thisUser = &user
		}
		privileges := user.DatasetPrivileges
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

	return nil
}

func (page *DeletePrivilege) Run() error {
	err := page.DB().DeleteDatasetPrivilege(models.UserDatasetPrivilege{
		UserID:    page.userId,
		DatasetID: page.datasetId,
	})

	if err != nil {
		return fmt.Errorf("%w: could not remove the privilege", handlers.GormToFiber(err))
	}

	return nil
}

func (page *DeletePrivilege) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}

type CreatePrivilege struct {
	handlers.ErrorHandler

	payload struct {
		Id            string
		UserId        string
		PrivilegeCode models.PrivilegeCode
	}

	datasetId     uint
	userId        uint
	privilegeCode models.PrivilegeCode
}

func (page *CreatePrivilege) Pre(c *fiber.Ctx) error {
	authUser, _ := c.Locals("AuthUser").(*models.User)
	if authUser == nil {
		return fiber.ErrUnauthorized
	}

	if err := c.BodyParser(&page.payload); err != nil {
		return fmt.Errorf("%w: %v", fiber.ErrBadRequest, err)
	}

	if datasetId, err := strconv.Atoi(page.payload.Id); err != nil {
		return fmt.Errorf(
			"%w: the provided id (%q) is not a positive integer",
			fiber.ErrBadRequest,
			page.payload.Id,
		)
	} else {
		page.datasetId = uint(datasetId)
	}

	if userId, err := strconv.Atoi(page.payload.UserId); err != nil {
		return fmt.Errorf(
			"%w: the provided userId (%q) is not a positive integer",
			fiber.ErrBadRequest,
			page.payload.UserId,
		)
	} else {
		page.userId = uint(userId)
	}

	page.privilegeCode = page.payload.PrivilegeCode

	dataset, err := page.DB().GetDataset(authUser, c.Params("team"), c.Params("dataset"))
	if err != nil {
		return handlers.GormToFiber(err)
	} else if dataset.ID != page.datasetId {
		return fiber.ErrBadRequest
	}

	if !authUser.CanManageDataset(*dataset) {
		return fiber.ErrForbidden
	}

	users, err := page.DB().GetUsersWithDatasetAccess(*dataset)
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
		if user.ID == page.userId {
			thisUser = &user
		}
		privileges := user.DatasetPrivileges
		if len(privileges) == 1 && privileges[0].Privilege.Code == models.AdminPrivilege {
			isAdmin = thisUser == &user
			adminUsers = append(adminUsers, user)
		}
	}

	if thisUser == nil {
		return fmt.Errorf("%w: we cannot find that user's privilege", fiber.ErrNotFound)
	} else if isAdmin && len(adminUsers) == 1 && thisUser.DatasetPrivileges[0].Privilege.Code != page.privilegeCode {
		return fmt.Errorf("%w: cannot remove the last admin user's admin privileges", fiber.ErrBadRequest)
	}

	return nil
}

func (page *CreatePrivilege) Run() error {
	err := page.DB().UpdateDatasetPrivilege(models.UserDatasetPrivilege{
		UserID:        page.userId,
		DatasetID:     page.datasetId,
		PrivilegeCode: page.privilegeCode,
	})

	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("%w: could not update the privilege", handlers.GormToFiber(err))
	}

	return nil
}

func (page *CreatePrivilege) RenderJson(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "success"})
}
