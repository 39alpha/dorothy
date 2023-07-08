package core

import (
	"bytes"
	"fmt"

	"github.com/kataras/iris/v12"
)

func ListOrganizations(ctx iris.Context) {
	config, ok := ctx.Values().Get("config").(*Config)
	if !ok {
		ctx.StopWithError(iris.StatusInternalServerError, fmt.Errorf("an internal error occured"))
		return
	}

	client, err := NewIpfs(config)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	orgs, err := client.GetOrganizations(ctx)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
	}

	results := []iris.Map{}
	for _, org := range orgs {
		results = append(results, iris.Map{"name": org.Name, "path": org.ToWebPath()})
	}

	ctx.JSON(results)
}

func CreateOrganization(ctx iris.Context) {
	config, ok := ctx.Values().Get("config").(*Config)
	if !ok {
		ctx.StopWithError(iris.StatusInternalServerError, fmt.Errorf("an internal error occured"))
		return
	}

	body, ok := ctx.Values().Get("jsonbody").(iris.Map)
	if !ok {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("Bad request body"))
		return
	}

	client, err := NewIpfs(config)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	org, ok := body["name"]
	if !ok {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf(`missing field: "org"`))
		return
	}

	path, err := client.CreateOrganization(ctx, org.(string))
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("an internal error occured"))
		return
	}

	ctx.JSON(iris.Map{"name": org, "path": path.ToWebPath()})
}

func ListRepositories(ctx iris.Context) {
	config, ok := ctx.Values().Get("config").(*Config)
	if !ok {
		ctx.StopWithError(iris.StatusInternalServerError, fmt.Errorf("an internal error occured"))
		return
	}

	client, err := NewIpfs(config)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	org := ctx.Params().Get("organization")

	repos, err := client.GetRepositories(ctx, org)
	if err != nil {
		ctx.StopWithStatus(iris.StatusNotFound)
		return
	}

	results := []iris.Map{}
	for _, repo := range repos {
		results = append(results, iris.Map{"name": repo.Name, "path": repo.ToWebPath()})
	}

	ctx.JSON(results)
}

func CreateRepository(ctx iris.Context) {
	config, ok := ctx.Values().Get("config").(*Config)
	if !ok {
		ctx.StopWithError(iris.StatusInternalServerError, fmt.Errorf("an internal error occured"))
		return
	}

	body, ok := ctx.Values().Get("jsonbody").(iris.Map)
	if !ok {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("Bad request body"))
		return
	}

	org := ctx.Params().Get("organization")

	client, err := NewIpfs(config)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	repo, ok := body["name"]
	if !ok {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf(`missing field: "repo"`))
		return
	}

	path, err := client.CreateRepository(ctx, org, repo.(string))
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("an internal error occured"))
		return
	}

	ctx.JSON(iris.Map{"name": repo, "path": path.ToWebPath()})
}

func GetManifest(ctx iris.Context) {
	config, ok := ctx.Values().Get("config").(*Config)
	if !ok {
		ctx.StopWithError(iris.StatusInternalServerError, fmt.Errorf("an internal error occured"))
		return
	}

	client, err := NewIpfs(config)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	org := ctx.Params().Get("organization")
	repo := ctx.Params().Get("repository")

	manifest, err := client.GetManifest(ctx, org, repo)
	if err != nil {
		ctx.StopWithStatus(iris.StatusNotFound)
		return
	}

	ctx.JSON(manifest)

}

func Push(ctx iris.Context) {
	config, ok := ctx.Values().Get("config").(*Config)
	if !ok {
		ctx.StopWithError(iris.StatusInternalServerError, fmt.Errorf("an internal error occured"))
		return
	}

	org := ctx.Params().Get("organization")
	repo := ctx.Params().Get("repository")

	body, err := ctx.GetBody()
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	buffer := new(bytes.Buffer)
	buffer.Write(body)

	manifest, err := ReadManifest(buffer)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	client, err := NewIpfs(config)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	merged, err := client.Commit(ctx, org, repo, manifest)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("an internal error occured"))
		return
	}

	ctx.JSON(merged)
}