package api

import (
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/tedsuo/rata"

	"crossent/micro/studio"
	"crossent/micro/studio/db"
	"crossent/micro/studio/client"
	"crossent/micro/studio/wrapper"
	"crossent/micro/studio/api/server"
)

func NewHandler(
	logger lager.Logger,
	repositoryFactory db.RepositoryFactory,
	uaa *client.UAA,
	wrapper wrappa.Wrappa,
) (http.Handler, error) {
	microServer := server.NewServer(
		logger,
		repositoryFactory,
		uaa,
		//authTokenGenerator,
		//csrfTokenGenerator,
		//expire,
	)
	composeServer := server.NewServer(
		logger,
		repositoryFactory,
		uaa,
	)

	handlers := map[string]http.Handler{
		studio.ListOrg: http.HandlerFunc(microServer.ListOrg),
		studio.ListOrgSpace: http.HandlerFunc(microServer.ListOrgSpace),
		studio.ListSpace: http.HandlerFunc(microServer.ListSpace),
		studio.ListApp: http.HandlerFunc(microServer.ListApp),
		studio.ListAppByEnv: http.HandlerFunc(microServer.ListAppByEnv),
		studio.ListAccess: http.HandlerFunc(microServer.ListAccess),
		studio.ListLink: http.HandlerFunc(microServer.ListLink),
		studio.ListConnect: http.HandlerFunc(microServer.ListConnect),
		studio.ListConnectSpace: http.HandlerFunc(microServer.ListConnectSpace),
		studio.CreateConnect: http.HandlerFunc(microServer.CreateConnect),
		studio.DeleteConnect: http.HandlerFunc(microServer.DeleteConnect),
		studio.ListConnectService: http.HandlerFunc(microServer.ListConnectService),
		studio.ListConnectServiceSpace: http.HandlerFunc(microServer.ListConnectServiceSpace),
		studio.CreateConnectService: http.HandlerFunc(microServer.CreateConnectService),
		studio.DeleteConnectService: http.HandlerFunc(microServer.DeleteConnectService),
		studio.ListServiceMarketplace: http.HandlerFunc(microServer.ListServiceMarketplace),

		studio.ListMicroservice: http.HandlerFunc(microServer.ListMicroservice),
		studio.CreateMicroservice: http.HandlerFunc(composeServer.CreateMicroservice),


		studio.GetMicroservice: http.HandlerFunc(microServer.GetMicroservice),
		studio.DeleteMicroservice: http.HandlerFunc(microServer.DeleteMicroservice),
		studio.GetMicroserviceLink: http.HandlerFunc(microServer.GetMicroserviceLink),
		studio.GetMicroserviceDetail: http.HandlerFunc(microServer.GetMicroserviceDetail),
		studio.GetMicroserviceComposition: http.HandlerFunc(composeServer.GetMicroserviceComposition),
		studio.UpdateMicroserviceComposition: http.HandlerFunc(composeServer.UpdateMicroserviceComposition),
		studio.UpdateMicroserviceState: http.HandlerFunc(composeServer.UpdateMicroserviceState),

		studio.Login: http.HandlerFunc(microServer.Login),
		studio.Logout: http.HandlerFunc(microServer.Logout),

		studio.ListMicroserviceApi: http.HandlerFunc(microServer.ListMicroserviceApi),
		studio.GetMicroserviceApi: http.HandlerFunc(microServer.GetMicroserviceApi),
		studio.SaveMicroserviceApi: http.HandlerFunc(microServer.SaveMicroserviceApi),

	}

	handler, err := rata.NewRouter(studio.Routes, wrapper.Wrap(handlers))
	if err != nil {
		panic("unable to create router: " + err.Error())
	}
	return wrappa.HttpWrap(handler), err
}


