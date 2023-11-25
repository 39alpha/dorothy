package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40

import (
	"context"

	"github.com/39alpha/dorothy/core"
	"github.com/39alpha/dorothy/core/model"
)

// CreateOrganization is the resolver for the createOrganization field.
func (r *mutationResolver) CreateOrganization(ctx context.Context, input model.NewOrganization) (*model.Organization, error) {
	return core.CreateOrganization(ctx, r.config, r.db, input)
}

// Organizations is the resolver for the organizations field.
func (r *queryResolver) Organizations(ctx context.Context) ([]*model.Organization, error) {
	return core.ListOrganizations(ctx, r.config, r.db)
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
