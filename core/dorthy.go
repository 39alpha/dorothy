package core

import (
	"context"

	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/recover"
)

type Dorthy struct {
	*iris.Application
	config *Config
}

func NewDorthy(config *Config) (*Dorthy, error) {
	session, err := NewDatabaseSession(config)
	if err != nil {
		return nil, err
	}

	app := iris.New()

	app.UseRouter(recover.New())
	app.UseRouter(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT"},
	}))

	app.UseGlobal(WithConfig(config))
	app.UseGlobal(WithDbSession(session))

	v0 := app.Party("v0", RecordBody)
	{
		v0.Get("/organization", ListOrganizations)
		v0.Put("/organization", ParseBody, CreateOrganization)
		v0.Get("/organization/{organization:string}/dataset", ListDatasets)
		v0.Put("/organization/{organization:string}/dataset", ParseBody, CreateDataset)
		v0.Get("/organization/{organization:string}/dataset/{dataset:string}", GetManifest)
		v0.Post("/organization/{organization:string}/dataset/{dataset:string}", Push)
	}

	dorothy := &Dorthy{app, config}
	err = dorothy.initialize()

	return dorothy, err
}

func (d *Dorthy) initialize() error {
	client, err := NewIpfs(d.config)
	if err != nil {
		return err
	}

	return client.FilesMkdir(context.TODO(), FS_ROOT, func(r *ipfs.RequestBuilder) error {
		r.Option("parents", true)
		return nil
	})
}
