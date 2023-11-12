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

func ListDatasets(ctx iris.Context) {
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

	datasets, err := client.GetDatasets(ctx, org)
	if err != nil {
		ctx.StopWithStatus(iris.StatusNotFound)
		return
	}

	results := []iris.Map{}
	for _, dataset := range datasets {
		results = append(results, iris.Map{"name": dataset.Name, "path": dataset.ToWebPath()})
	}

	ctx.JSON(results)
}

func CreateDataset(ctx iris.Context) {
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

	dataset, ok := body["name"]
	if !ok {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf(`missing field: "dataset"`))
		return
	}

	path, err := client.CreateDataset(ctx, org, dataset.(string))
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("an internal error occured"))
		return
	}

	ctx.JSON(iris.Map{"name": dataset, "path": path.ToWebPath()})
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
	dataset := ctx.Params().Get("dataset")

	manifest, err := client.GetManifest(ctx, org, dataset)
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
	dataset := ctx.Params().Get("dataset")

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

	merged, err := client.Commit(ctx, org, dataset, manifest)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, fmt.Errorf("an internal error occured"))
		return
	}

	ctx.JSON(merged)
}
