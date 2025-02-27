package server

import (
	"testing"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/server/model"
	"gorm.io/gorm/clause"
)

var session *DatabaseSession

func setup(t *testing.T) {
	var err error

	session, err = NewDatabaseSession(&core.DatabaseConfig{
		Path: ":memory:",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = session.Initialize(); err != nil {
		t.Fatal(err)
	}
}

func TestCanCreateUser(t *testing.T) {
	setup(t)

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

func TestCanCreateTeam(t *testing.T) {
	setup(t)

	team := &model.Team{
		Slug:        "team-0",
		Name:        "Team 0",
		Contact:     "39alpha@39alpharesearch.org",
		Description: "The team that started it all",
		IsPrivate:   true,
	}
	if result := session.Create(&team); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	var fetched model.Team
	result := session.First(&fetched, "teams.slug = ?", "team-0")
	if result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	if fetched.Slug != team.Slug {
		t.Errorf("expected team.Slug = %q; got %q", team.Slug, fetched.Slug)
	}

	if fetched.Name != team.Name {
		t.Errorf("expected team.Name = %q; got %q", team.Name, fetched.Name)
	}

	if fetched.Contact != team.Contact {
		t.Errorf("expected team.Contact = %v; got %v", team.Contact, fetched.Contact)
	}

	if fetched.Description != team.Description {
		t.Errorf("expected team.Description = %v; got %v", team.Description, fetched.Description)
	}

	if fetched.IsPrivate != team.IsPrivate {
		t.Errorf("expected team.IsPrivate = %v; got %v", team.IsPrivate, fetched.IsPrivate)
	}
}

func TestCanCreateDataset(t *testing.T) {
	setup(t)

	team := model.Team{Slug: "team0"}
	if result := session.Create(&team); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}
	if result := session.Where(&team).First(&team); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	dataset := &model.Dataset{
		Slug:        "scotus",
		Name:        "Supreme Court Opinion Analysis",
		Contact:     "39alpha@39alpharesearch.org",
		Description: "Some kind of crazy analysis of SCOTUS opinions",
		IsPrivate:   true,
		TeamID:      team.ID,
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

func TestUserTeamPrivileges(t *testing.T) {
	setup(t)

	team := &model.Team{Slug: "scotus"}
	if result := session.Create(team); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	users := []*model.User{
		{
			Email:    "39alpha@39alpharesearch.org",
			Name:     "39 Alpha Research",
			RoleCode: "admin",
			TeamPrivileges: []model.UserTeamPrivilege{
				{Team: team, PrivilegeCode: "admin"},
			},
		},
		{
			Email:    "doug@39alpharesearch.org",
			Name:     "Doug Moore",
			RoleCode: "user",
			TeamPrivileges: []model.UserTeamPrivilege{
				{Team: team, PrivilegeCode: "write"},
			},
		},
	}
	if result := session.Create(&users); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	users = []*model.User{}
	result := session.Preload("TeamPrivileges.Privilege").Preload("TeamPrivileges.Team").Find(&users)
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if len(users[0].TeamPrivileges) != 1 {
		t.Fatalf("expected 1 team privilege, got %d", len(users[0].TeamPrivileges))
	}
	if users[0].TeamPrivileges[0].Privilege.Code != "admin" {
		t.Fatalf("expected \"admin\" team privilege, got %q", users[0].TeamPrivileges[0].Privilege.Code)
	}
	if len(users[1].TeamPrivileges) != 1 {
		t.Fatalf("expected 1 team privilege, got %d", len(users[1].TeamPrivileges))
	}
	if users[1].TeamPrivileges[0].Privilege.Code != "write" {
		t.Fatalf("expected \"write\" team privilege, got %q", users[1].TeamPrivileges[0].Privilege.Code)
	}
}

func TestUserDatasetPrivileges(t *testing.T) {
	setup(t)

	team := &model.Team{Slug: "team-0"}
	if result := session.Create(team); result.Error != nil {
		t.Fatalf("%v", result.Error)
	}

	dataset := &model.Dataset{Slug: "dataset", Team: team}
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
		t.Fatalf("expected 1 team privilege, got %d", len(users[0].DatasetPrivileges))
	}
	if users[0].DatasetPrivileges[0].Privilege.Code != "admin" {
		t.Fatalf("expected \"admin\" team privilege, got %q", users[0].DatasetPrivileges[0].Privilege.Code)
	}
	if len(users[1].DatasetPrivileges) != 1 {
		t.Fatalf("expected 1 team privilege, got %d", len(users[1].DatasetPrivileges))
	}
	if users[1].DatasetPrivileges[0].Privilege.Code != "write" {
		t.Fatalf("expected \"write\" team privilege, got %q", users[1].DatasetPrivileges[0].Privilege.Code)
	}
}
