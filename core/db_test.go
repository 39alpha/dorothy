package core

import (
	"os"
	"testing"

	"github.com/39alpha/dorothy/core/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var session *DatabaseSession

func setup() {
	db, err := gorm.Open(sqlite.Open("dorothy_test.db?_foreign_keys=on&_ignore_check_constraints=off"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	session = &DatabaseSession{db}

	session.AutoMigrate(
		&model.Role{},
		&model.Privilege{},
		&model.Organization{},
		&model.Dataset{},
		&model.User{},
		&model.UserOrganizationPrivilege{},
		&model.UserDatasetPrivilege{},
	)

	roles := []*model.Role{
		{Code: "admin", Description: "The all-powerful entity"},
		{Code: "user", Description: "A standard user"},
	}
	if result := session.Create(&roles); result.Error != nil {
		panic(result.Error)
	}

	privileges := []*model.Privilege{
		{Code: "read", Description: "Read access"},
		{Code: "write", Description: "Write access"},
		{Code: "admin", Description: "Administrative access"},
	}
	if result := session.Create(&privileges); result.Error != nil {
		panic(result.Error)
	}
}

func teardown() {
	if err := os.Remove("dorothy_test.db"); err != nil {
		panic(err)
	}
}

func TestCanCreateUser(t *testing.T) {
	setup()
	defer teardown()

	user := &model.User{
		Email:        "39alpha@39alpharesearch.org",
		PasswordHash: []byte{},
		Name:         "39 Alpha Research",
		Orcid:        nil,
		Role:         &model.Role{Code: "admin"},
	}
	if result := session.Create(&user); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	var fetched model.User
	result := session.Preload(clause.Associations).First(&fetched, "users.email = ?", user.Email)
	if result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	if fetched.Email != user.Email {
		t.Errorf("expected user.Email = %q; got %q", user.Email, fetched.Email)
	}

	if fetched.Name != user.Name {
		t.Errorf("expected user.Name = %q; got %q", user.Name, fetched.Name)
	}

	if fetched.Orcid != user.Orcid {
		t.Errorf("expected user.Orcid = %v; got %v", user.Orcid, fetched.Orcid)
	}

	if fetched.Role.Code != user.Role.Code {
		t.Errorf("expected user.Role.Code = %q; got %q", user.Role.Code, fetched.Role.Code)
	}
}

func TestCanCreateOrganization(t *testing.T) {
	setup()
	defer teardown()

	org := &model.Organization{
		Slug:        "team-0",
		Name:        "Team 0",
		Contact:     "39alpha@39alpharesearch.org",
		Description: "The team that started it all",
		IsPrivate:   true,
	}
	if result := session.Create(&org); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	var fetched model.Organization
	result := session.First(&fetched, "organizations.slug = ?", "team-0")
	if result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	if fetched.Slug != org.Slug {
		t.Errorf("expected org.Slug = %q; got %q", org.Slug, fetched.Slug)
	}

	if fetched.Name != org.Name {
		t.Errorf("expected org.Name = %q; got %q", org.Name, fetched.Name)
	}

	if fetched.Contact != org.Contact {
		t.Errorf("expected org.Contact = %v; got %v", org.Contact, fetched.Contact)
	}

	if fetched.Description != org.Description {
		t.Errorf("expected org.Description = %v; got %v", org.Description, fetched.Description)
	}

	if fetched.IsPrivate != org.IsPrivate {
		t.Errorf("expected org.IsPrivate = %v; got %v", org.IsPrivate, fetched.IsPrivate)
	}
}

func TestCanCreateDataset(t *testing.T) {
	setup()
	defer teardown()

	org := model.Organization{Slug: "team0"}
	if result := session.Create(&org); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}
	if result := session.Where(&org).First(&org); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	dataset := &model.Dataset{
		Slug:           "scotus",
		Name:           "Supreme Court Opinion Analysis",
		Contact:        "39alpha@39alpharesearch.org",
		Description:    "Some kind of crazy analysis of SCOTUS opinions",
		IsPrivate:      true,
		OrganizationID: org.ID,
	}
	if result := session.Create(&dataset); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	var fetched model.Dataset
	result := session.First(&fetched, "datasets.slug = ?", "scotus")
	if result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	if fetched.Slug != dataset.Slug {
		t.Errorf("expected dataset.Slug = %q; got %q", dataset.Slug, fetched.Slug)
	}

	if fetched.Name != dataset.Name {
		t.Errorf("expected dataset.Name = %q; got %q", dataset.Name, fetched.Name)
	}

	if fetched.Contact != dataset.Contact {
		t.Errorf("expected dataset.Contact = %v; got %v", dataset.Contact, fetched.Contact)
	}

	if fetched.Description != dataset.Description {
		t.Errorf("expected dataset.Description = %v; got %v", dataset.Description, fetched.Description)
	}

	if fetched.IsPrivate != dataset.IsPrivate {
		t.Errorf("expected dataset.IsPrivate = %v; got %v", dataset.IsPrivate, fetched.IsPrivate)
	}
}

func TestUserOrganizationPrivileges(t *testing.T) {
	setup()
	defer teardown()

	org := &model.Organization{Slug: "scotus"}
	if result := session.Create(org); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	users := []*model.User{
		{
			Email:    "39alpha@39alpharesearch.org",
			Name:     "39 Alpha Research",
			RoleCode: "admin",
			OrganizationPrivileges: []model.UserOrganizationPrivilege{
				{Organization: org, PrivilegeCode: "admin"},
			},
		},
		{
			Email:    "doug@39alpharesearch.org",
			Name:     "Doug Moore",
			RoleCode: "user",
			OrganizationPrivileges: []model.UserOrganizationPrivilege{
				{Organization: org, PrivilegeCode: "write"},
			},
		},
	}
	if result := session.Create(&users); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	users = []*model.User{}
	result := session.Preload("OrganizationPrivileges.Privilege").Preload("OrganizationPrivileges.Organization").Find(&users)
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if len(users[0].OrganizationPrivileges) != 1 {
		t.Fatalf("expected 1 organization privilege, got %d", len(users[0].OrganizationPrivileges))
	}
	if users[0].OrganizationPrivileges[0].Privilege.Code != "admin" {
		t.Fatalf("expected \"admin\" organization privilege, got %q", users[0].OrganizationPrivileges[0].Privilege.Code)
	}
	if len(users[1].OrganizationPrivileges) != 1 {
		t.Fatalf("expected 1 organization privilege, got %d", len(users[1].OrganizationPrivileges))
	}
	if users[1].OrganizationPrivileges[0].Privilege.Code != "write" {
		t.Fatalf("expected \"write\" organization privilege, got %q", users[1].OrganizationPrivileges[0].Privilege.Code)
	}
}

func TestUserDatasetPrivileges(t *testing.T) {
	setup()
	defer teardown()

	org := &model.Organization{Slug: "team-0"}
	if result := session.Create(org); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	dataset := &model.Dataset{Slug: "dataset", Organization: org}
	if result := session.Create(dataset); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	users := []*model.User{
		{
			Email:    "39alpha@39alpharesearch.org",
			Name:     "39 Alpha Research",
			RoleCode: "admin",
			DatasetPrivileges: []model.UserDatasetPrivilege{
				{Dataset: dataset, PrivilegeCode: "admin"},
			},
		},
		{
			Email:    "doug@39alpharesearch.org",
			Name:     "Doug Moore",
			RoleCode: "user",
			DatasetPrivileges: []model.UserDatasetPrivilege{
				{Dataset: dataset, PrivilegeCode: "write"},
			},
		},
	}
	if result := session.Create(&users); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	users = []*model.User{}
	result := session.Preload("DatasetPrivileges.Privilege").Preload("DatasetPrivileges.Dataset").Find(&users)
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if len(users[0].DatasetPrivileges) != 1 {
		t.Fatalf("expected 1 organization privilege, got %d", len(users[0].DatasetPrivileges))
	}
	if users[0].DatasetPrivileges[0].Privilege.Code != "admin" {
		t.Fatalf("expected \"admin\" organization privilege, got %q", users[0].DatasetPrivileges[0].Privilege.Code)
	}
	if len(users[1].DatasetPrivileges) != 1 {
		t.Fatalf("expected 1 organization privilege, got %d", len(users[1].DatasetPrivileges))
	}
	if users[1].DatasetPrivileges[0].Privilege.Code != "write" {
		t.Fatalf("expected \"write\" organization privilege, got %q", users[1].DatasetPrivileges[0].Privilege.Code)
	}
}
