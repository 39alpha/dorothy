package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/39alpha/dorothy/core/model"
	"golang.org/x/crypto/bcrypt"
)

func CreateOrganization(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewOrganization) (*model.Organization, error) {
	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	org := &model.Organization{
		Slug:        input.Slug,
		Name:        input.Name,
		Contact:     input.Contact,
		Description: description,
	}

	if result := db.Create(org); result.Error != nil {
		return nil, result.Error
	}

	client, err := NewIpfs(config)
	if err != nil {
		return nil, err
	}

	_, err = client.CreateOrganization(ctx, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func ListOrganizations(ctx context.Context, config *Config, db *DatabaseSession) ([]*model.Organization, error) {
	var organizations []*model.Organization
	result := db.Find(&organizations)
	if result.Error != nil {
		return nil, result.Error
	}
	return organizations, nil
}

func GetOrganization(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetOrganization) (*model.Organization, error) {
	var organization model.Organization
	where := &model.Organization{}
	if input.ID != nil {
		where.ID = *input.ID
	}
	if input.Slug != nil && len(*input.Slug) > 0 {
		where.Slug = *input.Slug
	}
	if result := db.Where(where).First(&organization); result.Error != nil {
		return nil, result.Error
	}
	return &organization, nil
}

func CreateDataset(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewDataset) (*model.Dataset, error) {
	description := ""
	if input.Description != nil {
		description = *input.Description
	}

	dataset := &model.Dataset{
		Slug:           input.Slug,
		Name:           input.Name,
		Contact:        input.Contact,
		Description:    description,
		OrganizationID: input.OrganizationID,
	}

	if result := db.Create(dataset); result.Error != nil {
		return nil, result.Error
	}

	client, err := NewIpfs(config)
	if err != nil {
		return nil, err
	}

	_, err = client.CreateDataset(ctx, dataset)
	if err != nil {
		return nil, err
	}

	return dataset, nil
}

func ListDatasets(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetDatasets) ([]*model.Dataset, error) {
	var datasets []*model.Dataset
	result := db.
		Where(&model.Dataset{OrganizationID: input.OrganizationID}).
		First(&datasets)
	if result.Error != nil {
		return nil, result.Error
	}
	return datasets, nil
}

func GetDataset(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetDataset) (*model.Dataset, error) {
	var dataset model.Dataset
	result := db.Where(&model.Dataset{ID: input.ID}).First(&dataset)
	if result.Error != nil {
		return nil, result.Error
	}
	return &dataset, nil
}

func GetManifest(ctx context.Context, config *Config, db *DatabaseSession, input *model.Dataset) (*model.Manifest, error) {
	if input.Organization == nil {
		if result := db.Preload("Organizations").Where(input).First(input); result.Error != nil {
			return nil, fmt.Errorf("cannot find dataset")
		}
	}

	client, err := NewIpfs(config)
	if err != nil {
		return nil, err
	}

	manifest, err := client.GetManifest(ctx, input.Organization.Slug, input.Slug)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func GetUsers(ctx context.Context, config *Config, db *DatabaseSession) ([]*model.User, error) {
	var users []*model.User
	result := db.Select("id", "email", "name", "orcid").Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	return users, nil
}

func GetUser(ctx context.Context, config *Config, db *DatabaseSession, input *model.GetUser) (*model.User, error) {
	var user model.User
	result := db.
		Select("id", "email", "name", "orcid").
		Where(&model.User{Email: input.Email}).
		First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func CreateUser(ctx context.Context, config *Config, db *DatabaseSession, input *model.NewUser) (*model.User, error) {
	password_hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 8)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        input.Email,
		PasswordHash: password_hash,
		Name:         input.Name,
		Orcid:        input.Orcid,
	}
	if result := db.Create(user); result.Error != nil {
		return nil, result.Error
	}

	user = &model.User{Email: input.Email}
	if result := db.Where(&user).First(&user); result.Error != nil {
		return nil, result.Error
	}

	return user, nil
}

func Registration(config *Config, db *DatabaseSession) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var new_user model.NewUser
		if err := decoder.Decode(&new_user); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("illformed request"))
			return
		}

		_, err := CreateUser(r.Context(), config, db, &new_user)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("user already exists"))
			return
		}

		var user = model.User{
			Email: new_user.Email,
		}

		result := db.Select("id", "email", "name", "orcid").Where(&user).First(&user)
		if result.Error != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("an unexpected error occured"))
			return
		}

		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		if err := encoder.Encode(&user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("an unexpected error occured"))
			return
		}

		w.Write(buf.Bytes())
	}
}

func validateCredentials(db *DatabaseSession, email, password string) error {
	user := &model.User{
		Email: email,
	}

	result := db.Select("PasswordHash").Where(user).First(&user)
	if result.Error != nil || user.PasswordHash == nil {
		return fmt.Errorf("invalid email or password")
	}

	err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password))
	if err != nil {
		return fmt.Errorf("invalid email or password")
	}
	return nil
}

// func Login(tokenAuth *jwtauth.JWTAuth, config *Config, db *DatabaseSession) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         decoder := json.NewDecoder(r.Body)
//         var login model.UserLogin
//         if err := decoder.Decode(&login); err != nil {
//             w.WriteHeader(http.StatusBadRequest)
//             w.Write([]byte("illformed request"))
//             return
//         }

//         if err := validateCredentials(db, login.Email, login.Password); err != nil {
//             w.WriteHeader(http.StatusUnauthorized)
//             w.Write([]byte("invalid email or password"))
//             return
//         }

//         user := &model.User{Email: login.Email}
//         result := db.Select("id", "email", "name", "orcid").Where(user).First(user)
//         if result.Error != nil {
//             w.WriteHeader(http.StatusInternalServerError)
//             w.Write([]byte("an unexpected error occured"))
//             return
//         }

//         _, tokenString, err := tokenAuth.Encode(map[string]interface{}{
//             "id":    user.ID,
//             "email": user.Email,
//             "name":  user.Name,
//             "orcid": user.Orcid,
//         })
//         if err != nil {
//             w.WriteHeader(http.StatusInternalServerError)
//             w.Write([]byte("an unexpected error occured"))
//             return
//         }

//         response := map[string]string{
//             "token": tokenString,
//         }
//         var buf bytes.Buffer
//         encoder := json.NewEncoder(&buf)
//         if err := encoder.Encode(&response); err != nil {
//             w.WriteHeader(http.StatusInternalServerError)
//             w.Write([]byte("an unexpected error occured"))
//             return
//         }
//         w.Write(buf.Bytes())
//     }
// }
