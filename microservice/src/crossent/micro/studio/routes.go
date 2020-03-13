package studio

import "github.com/tedsuo/rata"

const (

	ListOrg = "ListOrg"
	ListManageOrg = "ListManageOrg"
	ListOrgSpace = "ListOrgSpace"
	ListSpace = "ListSpace"
	ListApp = "ListApp"
	ListAppByEnv = "ListAppByEnv"
	ListAccess = "ListAccess"
	ListLink = "ListLink"
	ListConnect = "ListConnect"
	ListConnectSpace = "ListConnectSpace"
	CreateConnect = "CreateConnect"
	DeleteConnect = "DeleteConnect"
	ListConnectService = "ListConnectService"
	ListConnectServiceSpace = "ListConnectServiceSpace"
	CreateConnectService = "CreateConnectService"
	DeleteConnectService = "DeleteConnectService"
	ListServiceMarketplace = "ListServiceMarketplace"

	ListMicroservice = "ListMicroservice"
	CreateMicroservice = "CreateMicroservice"

	GetMicroservice = "GetMicroservice"
	DeleteMicroservice = "DeleteMicroservice"
	GetMicroserviceLink = "GenMicroserviceLink"
	GetMicroserviceDetail = "GetMicroserviceDetail"
	GetMicroserviceComposition = "GetMicroserviceComposition"
	UpdateMicroserviceComposition = "UpdateMicroserviceComposition"
	UpdateMicroserviceState = "UpdateMicroserviceState"

	ListMicroserviceApi = "ListMicroserviceApi"
	GetMicroserviceApi = "GetMicroserviceApi"
	// --gy
	GetMicroserviceApiUserList = "GetMicroserviceApiUserList"

	GetMicroserviceApiSwagger = "GetMicroserviceApiSwagger"
	SaveMicroserviceApiSwagger = "SaveMicroserviceApiSwagger"
	GetMicroserviceApiRule = "GetMicroserviceApiRule"
	CreateMicroserviceApi = "CreateMicroserviceApi"
	CreateMicroserviceAppApi = "CreateMicroserviceAppApi"
	DeleteMicroserviceAppApi = "DeleteMicroserviceAppApi"
	UpdateMicroserviceAppApi = "UpdateMicroserviceAppApi"
	ListMicroserviceAppApiByMicroID = "ListMicroserviceAppApiByMicroID"
	ListMicroserviceAppApiByApiID = "ListMicroserviceAppApiByApiID"
	DeleteMicroserviceApi = "DeleteMicroserviceApi"
	ListMicroserviceFrontend = "ListMicroserviceFrontend"
	ListMicroserviceApiHealth = "ListMicroserviceApiHealth"
	GetMicroserviceNameCheck = "GetMicroserviceNameCheck"
	GetMicroservicePathCheck = "GetMicroservicePathCheck"

	//ListMicroserviceApi = "ListMicroserviceApi"
	//GetMicroserviceApi = "GetMicroserviceApi"
	//SaveMicroserviceApi = "SaveMicroserviceApi"

	Login = "Login"
	Logout = "Logout"

	GetMicroserviceMonitoring = "GetMicroserviceMonitoring"

)

var Routes = rata.Routes([]rata.Route{
	{Path: "/api/v1/orgs", Method: "GET", Name: ListOrg},
	{Path: "/api/v1/manageorgs", Method: "GET", Name: ListManageOrg},
	{Path: "/api/v1/orgs/:orgid/spaces", Method: "GET", Name: ListOrgSpace},
	{Path: "/api/v1/spaces", Method: "GET", Name: ListSpace},
	{Path: "/api/v1/apps", Method: "GET", Name: ListApp},
	{Path: "/api/v1/apps/env", Method: "GET", Name: ListAppByEnv},
	{Path: "/api/v1/accesses", Method: "GET", Name: ListAccess},
	{Path: "/api/v1/links", Method: "GET", Name: ListLink},
	{Path: "/api/v1/connects", Method: "GET", Name: ListConnect},
	{Path: "/api/v1/connects/:spaceid", Method: "GET", Name: ListConnectSpace},
	{Path: "/api/v1/connects/:spaceid", Method: "POST", Name: CreateConnect},
	{Path: "/api/v1/connects/:spaceid", Method: "DELETE", Name: DeleteConnect},
	{Path: "/api/v1/services", Method: "GET", Name: ListConnectService},
	{Path: "/api/v1/services/:spaceid", Method: "GET", Name: ListConnectServiceSpace},
	{Path: "/api/v1/services/:spaceid", Method: "POST", Name: CreateConnectService},
	{Path: "/api/v1/services/:spaceid", Method: "DELETE", Name: DeleteConnectService},
	{Path: "/api/v1/marketplace", Method: "GET", Name: ListServiceMarketplace},

	{Path: "/api/v1/microservices", Method: "GET", Name: ListMicroservice},
	{Path: "/api/v1/microservices", Method: "POST", Name: CreateMicroservice},

	{Path: "/api/v1/microservices/:id", Method: "GET", Name: GetMicroservice},
	{Path: "/api/v1/microservices/link/:id", Method: "GET", Name: GetMicroserviceLink},
	{Path: "/api/v1/microservices/detail/:id", Method: "GET", Name: GetMicroserviceDetail},
	{Path: "/api/v1/microservices/:id/composition", Method: "GET", Name: GetMicroserviceComposition},
	{Path: "/api/v1/microservices/:id/composition", Method: "PUT", Name: UpdateMicroserviceComposition},
	{Path: "/api/v1/microservices/:id/state", Method: "PUT", Name: UpdateMicroserviceState},

	{Path: "/api/v1/login", Method: "POST", Name: Login},
	{Path: "/api/v1/logout", Method: "POST", Name: Logout},

	{Path: "/api/v1/apigateway", Method: "GET", Name: ListMicroserviceApi},
	{Path: "/api/v1/apigateway", Method: "POST", Name: CreateMicroserviceApi},
	{Path: "/api/v1/apigateway/:id", Method: "GET", Name: GetMicroserviceApi},
	{Path: "/api/v1/apigateway/:id/testb", Method: "GET", Name: ListMicroserviceAppApiByApiID},
	{Path: "/api/v1/apigateway/:id/rule", Method: "GET", Name: GetMicroserviceApiRule},
	{Path: "/api/v1/apigateway/:id/swagger", Method: "GET", Name: GetMicroserviceApiSwagger},
	{Path: "/api/v1/apigateway/:id/swagger", Method: "POST", Name: SaveMicroserviceApiSwagger},
	{Path: "/api/v1/apigateway/:id/api", Method: "GET", Name: ListMicroserviceAppApiByMicroID},
	{Path: "/api/v1/apigateway/:id/api", Method: "POST", Name: CreateMicroserviceAppApi},
	{Path: "/api/v1/apigateway/:id/api", Method: "PUT", Name: UpdateMicroserviceAppApi},
	{Path: "/api/v1/apigateway/:id/api", Method: "DELETE", Name: DeleteMicroserviceAppApi},
	{Path: "/api/v1/apigateway/:id", Method: "DELETE", Name: DeleteMicroserviceApi},
	{Path: "/api/v1/apigateway/frontend/microservices", Method: "GET", Name: ListMicroserviceFrontend},
	{Path: "/api/v1/apigateway/health/microservices", Method: "GET", Name: ListMicroserviceApiHealth},
	{Path: "/api/v1/apigateway/name/check", Method: "GET", Name: GetMicroserviceNameCheck},
	{Path: "/api/v1/apigateway/path/check", Method: "GET", Name: GetMicroservicePathCheck},

	//{Path: "/api/v1/microservices/api/list", Method: "GET", Name: ListMicroserviceApi},
	//{Path: "/api/v1/microservices/:id/api", Method: "GET", Name: GetMicroserviceApi},
	//{Path: "/api/v1/microservices/:id/api", Method: "PUT", Name: SaveMicroserviceApi},
	{Path: "/api/v1/microservices/:id", Method: "DELETE", Name: DeleteMicroservice},

	{Path: "/api/v1/monitoring/:id", Method: "GET", Name: GetMicroserviceMonitoring},
})
