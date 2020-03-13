package server

import (
	"bytes"
	"code.cloudfoundry.org/lager"
	"crossent/micro/studio/client"
	"crossent/micro/studio/domain"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type T interface{}

func (s *Server) CreateMicroservice(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("compose")
	logger.Debug("CreateMicroservice")

	var request domain.ComposeRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Error("Decode err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}


	if count, err := s.repositoryFactory.Compose().ListMicroserviceByName(request.Name); err == nil {
		if count > 0 {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, "duplicateName", http.StatusInternalServerError)
			return
		}
	}


	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	request.Status = domain.STATUS_INITIAL
	request.UserId = userid

	// 1. create Microservice
	id, err := s.repositoryFactory.Compose().CreateMicroservice(request)
	if err != nil {
		logger.Error("CreateMicroservice err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 2-1. Services
	for _, service := range request.Services.Resources {
		serviceRequest := domain.MicroserviceService{MicroID: id, ServiceGuid: service.Meta.Guid}

		// 2-1-1. CF Creating a Service Instance
		data := domain.ServiceInstance {
			Name: fmt.Sprintf("%s-%s", service.Entity.Name, request.Name),
			ServicePlanGuid: service.Entity.ServicePlanGuid,
			SpaceGuid: request.SpaceGuid,
			Parameters: map[string]interface{}{"target_org": request.OrgName, "target_space": request.SpaceName},
		}

		var serviceInstance ServiceResource
		serviceInstance, err = s.CreateServiceInstance(data)
		if err != nil {
			logger.Error("CreateServiceInstance err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 2-1-2. Insert db
		serviceRequest.ServiceGuid = serviceInstance.Metadata.GUID
		_, err := s.repositoryFactory.Compose().CreateMicroserviceService(serviceRequest)
		if err != nil {
			logger.Error("CreateMicroserviceService err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		name := fmt.Sprintf("%sapp%s", strings.Split(service.Entity.Name, "-")[0], serviceInstance.Metadata.GUID)
		if service.Entity.Name == domain.SERVICE_CONFIG_SERVER {
			s.saveEssentialApp(id, name)
		} else {
			go s.saveEssentialApp(id, name)
		}
	}
	// 2-2. Apps
	// 2-2-1. Insert gatewayapp to DB
	app, _ := s.GetAppByName(domain.MSA_GATEWAY_APP)
	appRequest := domain.MicroserviceApp{MicroID: id, SourceGuid: app.Resources[0].Meta.Guid}
	_, err = s.repositoryFactory.Compose().CreateMicroserviceApp(appRequest)
	if err != nil {
		logger.Error("CreateMicroserviceApp err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 2-2-2. Insert sample app to DB
	for _, app := range request.Apps.Resources {
		sourceAppGuid := app.Meta.Guid
		// 2-2-1. Insert db
		appRequest := domain.MicroserviceApp{MicroID: id, SourceGuid: sourceAppGuid}
		_, err = s.repositoryFactory.Compose().CreateMicroserviceApp(appRequest)
		if err != nil {
			logger.Error("CreateMicroserviceApp err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 3. get microservice
	micro, err := s.repositoryFactory.View().GetMicroservice(id)
	if err != nil {
		logger.Error("GetMicroservice err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(micro)
	if err != nil {
		logger.Error("Marshal err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
func (s *Server) saveEssentialApp(microID int, name string) {
	logger := s.logger.Session("compose")
	logger.Debug("saveEssentialApp")

	var app domain.Apps
	for i := 1; i < 5; i++ {
		time.Sleep(time.Millisecond * time.Duration(500 * i))
		app, _ = s.GetAppByName(name)
		if app.Count > 0 {
			break
		}
	}
	if app.Count > 0 {
		var essential string
		if strings.HasPrefix(app.Resources[0].Entity.Name, domain.MSA_REGISTRY_APP) {
			essential = domain.REGISTRY_NAME
		} else if strings.HasPrefix(app.Resources[0].Entity.Name, domain.MSA_CONFIG_APP) {
			essential = domain.CONFIG_NAME
		}
		guid := app.Resources[0].Meta.Guid
		appRequest := domain.MicroserviceApp{MicroID: microID, AppGuid: guid, SourceGuid: guid, Essential: essential}
		_, err := s.repositoryFactory.Compose().CreateMicroserviceApp(appRequest)
		if err != nil {
			logger.Error("[saveEssentialApp] CreateMicroserviceApp err >>>", err)
		}
	}
}

func (s *Server) GetMicroserviceComposition(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("compose")
	logger.Debug("GetMicroserviceComposition")

	token, err := s.uaa.GetAuthToken()
	if err != nil {
		logger.Error("failed cf get auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	id := r.FormValue(":id")
	idInt, _ := strconv.Atoi(id)
	msa, err := s.repositoryFactory.View().GetMicroservice(idInt)
	if err != nil {
		logger.Error("failed GetMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	// microservice 상세정보
	apps := []domain.AppResource{}
	services := []domain.ServiceInstanceResource{}
	bindings := []domain.ServiceBinding{}
	routes := []route{}
	var routingData []byte
	networkPolicyApps := []string{}
	properties := []property{}
	var configUrl string

	// apps
	msApps, err := s.repositoryFactory.Compose().ListMicroserviceAppApp(msa.ID)
	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	for _, msApp := range msApps {
		app := domain.AppResource{}
		// App 정보
		if msa.Status == domain.STATUS_INITIAL || msApp.AppGuid == "" {
			app.Meta.Guid = msApp.SourceGuid
		} else {
			app.Meta.Guid = msApp.AppGuid
			networkPolicyApps = append(networkPolicyApps, msApp.AppGuid)
		}
		summary := s.GetAppSummary(app.Meta.Guid, token)

		// routing 정보
		if strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) {
			route := summary.Routes[0]
			// configapp basic auth
			basicAuth := fmt.Sprintf("%s:%s", summary.Environment["basic-user"], summary.Environment["basic-secret"])
			configUrl = fmt.Sprintf("http://%s@%s.%s", basicAuth, route.Host, route.Domain.Name)
			resp, err := http.Get(fmt.Sprintf("%s/config/read/apigateway?refresh=true", configUrl))
			if err != nil {
				logger.Error("failed http get apigateway", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if err != nil {
				logger.Error("failed http get apigateway response", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}
			if resp.StatusCode == http.StatusOK {
				routingData = body
			}
		}

		if strings.HasPrefix(summary.Name, domain.MSA_REGISTRY_APP) || strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) {
			continue
		}
		if msa.Status == domain.STATUS_INITIAL || msApp.AppGuid == "" {
			app.Meta.Guid = domain.STATUS_INITIAL + "_" + summary.Name
			app.Entity.Name = fmt.Sprintf("%s-%s", summary.Name, msa.Name)
		} else {
			app.Meta.Guid = summary.Guid
			app.Entity.Name = summary.Name
			app.Entity.Instances = summary.Instances
			app.Entity.Memory = summary.Memory
			app.Entity.DiskQuota = summary.DiskQuota
		}
		apps = append(apps, app)

		// service binding 정보
		if msa.Status != domain.STATUS_INITIAL && msApp.AppGuid != "" {
			for _, service := range summary.Services {
				var binding domain.ServiceBinding
				binding.AppGuid = app.Meta.Guid
				binding.ServiceInstanceGuid = service.Guid
				bindings = append(bindings, binding)
			}
		}
	}

	// services
	msServices, err := s.repositoryFactory.Compose().ListMicroserviceAppService(msa.ID)
	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	for _, msService := range msServices {
		service := domain.ServiceInstanceResource{}
		serviceInstance := s.GetServiceInstanceByGuid(msService.ServiceGuid)
		service.Meta.Guid = serviceInstance.Metadata.GUID
		service.Entity.Name = serviceInstance.Entity.Name
		service.Entity.ServicePlanGuid = serviceInstance.Entity.Service_plan_guid
		services = append(services, service)
	}

	// routings
	if routingData != nil {
		keys := make(map[string]string)
		m := make(map[string]string)

		result := strings.Split(string(routingData), "\n")

		for _, s := range result {
			if strings.Trim(s, " ") != "" {
				sp := strings.Split(s, ".")
				if _, exist := keys[sp[2]]; !exist {
					keys[sp[2]] = sp[2]
				}
				spm := strings.Split(s, "=")
				m[spm[0]] = spm[1]
			}
		}

		for k := range keys {
			if serviceId, exist := m[fmt.Sprintf("zuul.routes.%s.serviceId", k)]; exist {
				if path, exist2 := m[fmt.Sprintf("zuul.routes.%s.path", k)]; exist2 {
					r := route{}
					r.ServiceName = serviceId
					r.Path = path
					routes = append(routes, r)
				}
			}
		}
	}

	// cf network-policies
	ids := strings.Join(networkPolicyApps[:], ",")
	access, _ := s.GetAccessById(ids)

	// properties
	for _, app := range apps {
		resp, err := http.Get(fmt.Sprintf("%s/config/read/%s?refresh=true", configUrl, app.Entity.Name))
		if err != nil {
			logger.Error("failed http get config", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			logger.Error("failed http get config response", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}
		if code := resp.StatusCode; code < 200 || code > 299 {
			body = nil
		}
		if body != nil {
			result := strings.Split(string(body), "\n")

			for _, s := range result {
				if strings.Trim(s, " ") != "" {
					p := property{}
					p.AppName = app.Entity.Name
					p.Properties = s
					properties = append(properties, p)
				}
			}
		}
	}

	type microserviceComposition struct {
		Microservice domain.Compose `json:"microservice"`
		Apps []domain.AppResource `json:"apps"`
		Services []domain.ServiceInstanceResource `json:"services"`
		Bindings []domain.ServiceBinding `json:"bindings"`
		Routes []route `json:"routes"`
		Policies []Policies `json:"policies"`
		Registries []registry `json:"registries"`
		Properties []property `json:"properties"`
	}

	details := microserviceComposition{
		Apps: apps,
		Services: services,
		Bindings: bindings,
		Routes: routes,
		Policies: access.Policies,
		Properties: properties,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)


}

func (s *Server) UpdateMicroserviceComposition(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("compose")
	logger.Debug("UpdateMicroserviceComposition")

	realIds := make(map[string]interface{})

	var request domain.ComposeRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Error("Decode err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	composition := request.Composition


	session := domain.SessionManager.Load(r)
	token := &client.CF_TOKEN{}
	accessToken, err := session.GetString(domain.UAA_TOKEN_NAME)
	if accessToken == "" {
		token, err = s.uaa.GetAuthToken()
	} else {
		token.AccessToken = accessToken
	}
	adminToken, err := s.uaa.GetAuthToken()
	if err != nil {
		logger.Error("failed cf get auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	cf, err := s.uaa.CfClient()
	if err != nil {
		logger.Error("failed cf client", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	// get shared domains
	sharedDomains, err := s.ListSpaceDomains(request.SpaceGuid)
	if err != nil {
		logger.Error("[UpdateMicroserviceComposition] ListSpaceDomains err >>>", err)
		return
	}
	sharedDomain := sharedDomains.Resources[0]
	frontUrl := ""

	var configappName string
	// 2. create INITIAL services (db, cf)
	for _, service := range composition.Services.Resources {
		if strings.Index(service.Entity.Name, domain.SERVICE_CONFIG_SERVER) > -1 {
			configappName = fmt.Sprintf("configapp%s", service.Meta.Guid)
		}
		if strings.Index(service.Meta.Guid, domain.STATUS_INITIAL) > -1 {
			serviceId := service.Meta.Guid[8:]
			serviceRequest := domain.MicroserviceService{MicroID: request.ID, ServiceGuid: serviceId}

			// 2-1. CF Creating a Service Instance
			data := domain.ServiceInstance {
				//Name: fmt.Sprintf("%s-%s", service.Entity.Name, request.Name),
				Name: fmt.Sprintf("%s", service.Entity.Name),
				ServicePlanGuid: service.Entity.ServicePlanGuid,
				SpaceGuid: request.SpaceGuid,
			}
			result, err := s.CreateServiceInstance(data)
			if err != nil {
				logger.Error("[UpdateMicroserviceComposition] CreateServiceInstance err >>>", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 2-2. Insert db
			serviceRequest.ServiceGuid = result.Metadata.GUID
			_, err = s.repositoryFactory.Compose().CreateMicroserviceService(serviceRequest)
			if err != nil {
				logger.Error("[UpdateMicroserviceComposition] CreateMicroserviceService err >>>", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			realIds[service.Meta.Guid] = serviceRequest.ServiceGuid
		}
	}
	// 3. create INITIAL apps (db, cf)
	// 3-1. get inserted ms apps
	msAppList, err := s.repositoryFactory.Compose().ListMicroserviceAppApp(request.ID)
	msApps := make(map[string]interface{}, len(msAppList))
	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	for _, app := range msAppList {
		msApps[app.SourceGuid] = app.MicroID
	}
	configAppEnv := make(map[string]interface{})
	for _, app := range composition.Apps.Resources {
		if strings.HasPrefix(app.Meta.Guid, domain.STATUS_INITIAL) {
			sourceAppGuid := app.Meta.Guid[8:]
			if sourceAppGuid == domain.SAMPLE_APP_FRONT || sourceAppGuid == domain.SAMPLE_APP_BACK || sourceAppGuid == domain.MSA_GATEWAY_APP {
				app, err := s.GetAppByName(sourceAppGuid)
				if err != nil {
					logger.Error("[saveGatewayApp] GetAppByName err >>>", err)
					return
				}
				sourceAppGuid = app.Resources[0].Meta.Guid
			}
			summary := s.GetAppSummary(sourceAppGuid, adminToken)

			// 3-2. CF Creating an App
			data := domain.App {
				//Name: fmt.Sprintf("%s-%s", app.Entity.Name, request.Name),
				Name: fmt.Sprintf("%s", app.Entity.Name),
				Instances: summary.Instances,
				Memory: summary.Memory,
				DiskQuota: summary.DiskQuota,
				State: domain.APP_STATE_STOPPED,
				SpaceGuid: request.SpaceGuid,
				//Environment: map[string]interface{}{"msa": "true"},
			}
			//if strings.Index(app.Entity.Name, domain.SAMPLE_APP_FRONT) > -1 {
			//	data.Environment = map[string]interface{}{"back.name": fmt.Sprintf("back-%s", request.Name)}
			//}
			if strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) {
				configAppEnv["basic-user"] = summary.Environment["basic-user"]
				configAppEnv["basic-secret"] = summary.Environment["basic-secret"]
			}
			if strings.HasPrefix(summary.Name, domain.MSA_GATEWAY_APP) {
				if len(configAppEnv) == 0 {
					tmpApp, err := s.GetAppByName(configappName)
					if err != nil {
						logger.Error("[saveGatewayApp] GetAppByName err >>>", err)
						return
					}
					configAppEnv["basic-user"] = tmpApp.Resources[0].Entity.Environment["basic-user"].(string)
					configAppEnv["basic-secret"] = tmpApp.Resources[0].Entity.Environment["basic-secret"].(string)
				}
				data.Environment = configAppEnv
			}
			createdApp, err := s.CreateApp(data, token)
			if err != nil {
				logger.Error("[UpdateMicroserviceComposition] CreateApp err >>>", err)
				jsonStr, _ := json.Marshal(data)
				fmt.Println("[UpdateMicroserviceComposition] CreateApp data >>>", string(jsonStr))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			createAppGuid := createdApp.Meta.Guid
			// 3-3. CF Copy the app bits for an App
			m2 := map[string]string{
				"source_app_guid" : sourceAppGuid,
			}
			_, err = s.CopyAppBits(createAppGuid, m2)
			if err != nil {
				logger.Error("[UpdateMicroserviceComposition] CopyAppBits err >>>", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// front app
			if app.Entity.State == domain.SAMPLE_APP_FRONT {
				// 3-4. CF Creating a Route
				routeData := domain.Route{
					Host: createdApp.Entity.Name,
					DomainGuid: sharedDomain.Meta.Guid,
					SpaceGuid: request.SpaceGuid,
				}
				route, err := s.CreateRoute(routeData)
				if err != nil {
					logger.Error("[UpdateMicroserviceComposition] CreateRoute err >>>", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				// 3-5. CF Associate Route with the App
				_, err = s.AssociateRoute(createAppGuid, route.Meta.Guid)
				if err != nil {
					logger.Error("[UpdateMicroserviceComposition] AssociateRoute err >>>", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				frontUrl = fmt.Sprintf("%s.%s", route.Entity.Host, sharedDomain.Entity.Name)
			}
			// gateway app route
			if strings.HasPrefix(app.Entity.Name, domain.MSA_GATEWAY_APP) {
				// 3-4. CF Creating a Route
				routeData := domain.Route{
					Host: createdApp.Entity.Name,
					DomainGuid: sharedDomain.Meta.Guid,
					SpaceGuid: request.SpaceGuid,
				}
				route, err := s.CreateRoute(routeData)
				if err != nil {
					logger.Error("[UpdateMicroserviceComposition] CreateRoute gateway err >>>", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				// 3-5. CF Associate Route with the App
				_, err = s.AssociateRoute(createAppGuid, route.Meta.Guid)
				if err != nil {
					logger.Error("[UpdateMicroserviceComposition] AssociateRoute gateway err >>>", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			}
			// 3-6. Insert db
			appRequest := domain.MicroserviceApp{MicroID: request.ID, AppGuid: createAppGuid, SourceGuid: sourceAppGuid, Essential: app.Entity.State}
			if msApps[sourceAppGuid] != nil {
				_, err = s.repositoryFactory.Compose().UpdateMicroserviceApp(appRequest)
			} else {
				_, err = s.repositoryFactory.Compose().CreateMicroserviceApp(appRequest)
			}
			if err != nil {
				logger.Error("[UpdateMicroserviceComposition] Create of Update MicroserviceApp err >>>", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			realIds[app.Meta.Guid] = createAppGuid
		}
	}
	// 4. bind app - service (cf)
	for _, binding := range composition.ServiceBindings.Resources {
		appGuid := binding.Entity.AppGuid
		if strings.Index(appGuid, domain.STATUS_INITIAL) > -1 {
			appGuid = realIds[binding.Entity.AppGuid].(string)
		}
		serviceInstanceGuid := binding.Entity.ServiceInstanceGuid
		if strings.Index(serviceInstanceGuid, domain.STATUS_INITIAL) > -1 {
			serviceInstanceGuid = realIds[binding.Entity.ServiceInstanceGuid].(string)
		}
		b := BindingService{}
		b.App_guid = appGuid
		b.Service_instance_guid = serviceInstanceGuid

		isExistServiceInstance := false
		if sb, err := s.ListServiceBindingByApp(appGuid); err == nil {
			for _, sbr := range sb.Resources {
				if sbr.Entity.ServiceInstanceGuid == serviceInstanceGuid {
					isExistServiceInstance = true
					break
				}
			}
		}

		if !isExistServiceInstance {
			_, err := s.CreateBinding(b)
			if err != nil {
				var cfErrBody domain.CloudFoundryErrBody
				json.Unmarshal([]byte(err.Error()), &cfErrBody)
				if cfErrBody.Message == "The app is already bound to the service." {
					continue
				}
				logger.Error("[UpdateMicroserviceComposition] CreateBinding err >>>", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	// 5-1. delete network policy (cf)
	if len(composition.DelPolicies) > 0 {
		networkPolicies := domain.Policies{}
		policies := []domain.Policy{}
		for _, policy := range composition.DelPolicies {
			if strings.Index(policy.Source.ID, domain.STATUS_INITIAL) > -1 {
				policy.Source.ID = realIds[policy.Source.ID].(string)
			}
			if strings.Index(policy.Destination.ID, domain.STATUS_INITIAL) > -1 {
				policy.Destination.ID = realIds[policy.Destination.ID].(string)
			}
			policies = append(policies, policy)
		}
		networkPolicies.TotalPolicies = len(composition.Policies)
		networkPolicies.Policies = policies

		err = s.DeleteAccess(networkPolicies)
		if err != nil {
			logger.Error("[UpdateMicroserviceComposition] DeleteAccess err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// 5-2. add network policy (cf)
	if len(composition.Policies) > 0 {
		networkPolicies := domain.Policies{}
		policies := []domain.Policy{}
		for _, policy := range composition.Policies {
			if strings.Index(policy.Source.ID, domain.STATUS_INITIAL) > -1 {
				policy.Source.ID = realIds[policy.Source.ID].(string)
			}
			if strings.Index(policy.Destination.ID, domain.STATUS_INITIAL) > -1 {
				policy.Destination.ID = realIds[policy.Destination.ID].(string)
			}
			policies = append(policies, policy)
		}
		networkPolicies.TotalPolicies = len(composition.Policies)
		networkPolicies.Policies = policies
		err = s.CreateAccess(networkPolicies)
		if err != nil {
			logger.Error("[UpdateMicroserviceComposition] CreateAccess err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Delete App
	for _, delapp := range composition.DelApps.Resources {
		if strings.HasPrefix(delapp.Entity.Name, domain.MSA_GATEWAY_APP) == false {
			//fmt.Println(delapp)
			if sb, err := s.ListServiceBindingByApp(delapp.Meta.Guid); err == nil {
				for _, sbr := range sb.Resources {
					// delete service binding
					if err := s.DeleteServiceBindingByApp(delapp.Meta.Guid, sbr.Meta.Guid); err != nil {
						logger.Error("DeleteMicroservice Servicebindingg error", err, lager.Data{"appGUID": delapp.Meta.Guid})
					}
				}
			}
			// delete route
			if listroutes, err := s.ListRouteForApp(delapp.Meta.Guid); err == nil {
				for _, route := range listroutes.Resources {
					if err := s.DeleteRoute(route.Meta.Guid); err != nil {
						logger.Error("DeleteRouteForApp error", err, lager.Data{"appGUID": delapp.Meta.Guid})
					}
				}
			}


			// delete app
			if err := cf.DeleteApp(delapp.Meta.Guid); err != nil {
				logger.Error("DeleteMicroservice error", err, lager.Data{"appGUID": delapp.Meta.Guid})
			}

			appRequest := domain.MicroserviceApp{MicroID: request.ID, AppGuid: delapp.Meta.Guid, SourceGuid: ""}
			err = s.repositoryFactory.Compose().DeleteMicroserviceApp(appRequest)
			if err != nil {
				logger.Error("[UpdateMicroserviceComposition] Delete MicroserviceApp err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	//  Delete Service
	for _, delservice := range composition.DelServices.Resources {
		// delete service binding
		//if err := s.DeleteServiceBinding(delservice.Meta.Guid); err != nil {
		//	logger.Error("DeleteMicroservice DeleteServiceBinding error", err, lager.Data{"appGUID": delservice.Meta.Guid})
		//}

		// delete service instance (recursive)
		if err := s.DeleteServiceInstanceByGuid(delservice.Meta.Guid); err != nil {
			logger.Error("DeleteMicroservice Serviceinstance error", err, lager.Data{"appGUID": delservice.Meta.Guid})
		}

		serviceRequest := domain.MicroserviceService{MicroID: request.ID, ServiceGuid: delservice.Meta.Guid}
		err = s.repositoryFactory.Compose().DeleteMicroserviceService(serviceRequest)
		if err != nil {
			logger.Error("[UpdateMicroserviceComposition] Delete MicroserviceApp err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 1. UpdateMicroserviceStatus
	appRequest := domain.ComposeRequest{ID: request.ID, Status: request.Status, Version: request.Version, Visible: request.Visible, Url: frontUrl }
	_, err = s.repositoryFactory.Compose().UpdateMicroservice(appRequest)
	if err != nil {
		logger.Error("[UpdateMicroserviceState] UpdateMicroserviceStatus err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. add route (configapp)
	// check configapp status: running
	configapps, err := s.GetAppByName(configappName)
	if err != nil {
		logger.Error("[UpdateMicroserviceComposition] GetAppByName err >>>", err)
		return
	}
	configapp := configapps.Resources[0]
	if configapp.Entity.State == domain.APP_STATE_STOPPED {
		body := map[string]string{ "state" : domain.APP_STATE_STARTED, }
		_, err = s.UpdateApp(configapp.Meta.Guid, body, adminToken)
		if err != nil {
			logger.Error("[UpdateMicroserviceComposition] UpdateApp err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	var jsonData []string
	for _, route := range composition.Routes {
		if route["service"].(string) != "" {
			jsonData = append(jsonData, fmt.Sprintf(`{"zuul.routes.%s.serviceId":"%s"}`,route["service"].(string),route["service"].(string)))
			jsonData = append(jsonData, fmt.Sprintf(`{"zuul.routes.%s.path":"%s"}`,route["service"].(string),route["path"].(string)))
		}
	}
	var jsonStr = []byte(`[`+strings.Join(jsonData,",")+`]`)
	basicUser := configapp.Entity.Environment["basic-user"]
	basicSecret := configapp.Entity.Environment["basic-secret"]
	endpoint := fmt.Sprintf("http://%s:%s@%s.%s/config/write/apigateway", basicUser, basicSecret, configappName, sharedDomain.Entity.Name)
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	response, err := client.Do(req)
	if err != nil {
		logger.Error("[UpdateMicroserviceComposition] add route err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	// 7. add config (configapp)
	for _, config := range composition.Configs {
		property, err := json.Marshal(config["property"])
		if err != nil {
			logger.Error("[UpdateMicroserviceComposition] json marshal err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		endpoint := fmt.Sprintf("http://%s:%s@%s.%s/config/write/%s", basicUser, basicSecret, configappName, sharedDomain.Entity.Name, config["app"])
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(property))
		req.Header.Set("Content-Type", "application/json")

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		response, err := client.Do(req)
		if err != nil {
			logger.Error("[UpdateMicroserviceComposition] add config err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer response.Body.Close()

	}

	// Refresh properties for all app.
	if err := refreshProperties(fmt.Sprintf("%s", basicUser), fmt.Sprintf("%s", basicSecret), configappName, sharedDomain.Entity.Name); err != nil {
		logger.Error("[UpdateMicroserviceComposition] refreshProperties err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func refreshProperties(basicUser, basicSecret, configappName, sharedDomain string) error {
	// Refresh properties for all app.
	endpoint := fmt.Sprintf("http://%s:%s@%s.%s/refresh", basicUser, basicSecret, configappName, sharedDomain)
	var jsonStr = []byte("")
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("[UpdateMicroserviceComposition] add config err: %s ", err.Error())
	}

	defer response.Body.Close()

	return nil
}

func (s *Server) UpdateMicroserviceState(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("compose")
	logger.Debug("UpdateMicroserviceState")

	strId := r.FormValue(":id")
	id, _ := strconv.Atoi(strId)
	var request domain.ComposeRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		logger.Error("[UpdateMicroserviceState] Decode err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := s.uaa.GetAuthToken()
	if err != nil {
		logger.Error("[UpdateMicroserviceState] failed cf get auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	// 1. start/stop ms app
	// 1-1. configapp, registryapp
	query := fmt.Sprintf("name+IN+config-server-%s,registry-server-%s", request.Name, request.Name)
	serviceInstances, err := s.GetServiceInstanceByQuery(query)
	configApp := domain.AppResource{}
	maxCount := 5
	for _, si := range serviceInstances.Resources {
		appName := fmt.Sprintf("%sapp%s", strings.Split(si.Entity.Name, "-")[0], si.Metadata.GUID)
		apps, _ := s.GetAppByName(appName)
		app := apps.Resources[0]
		// spring-cloud-config가 정상적으로 동작하는지 확인하고 spring-cloud-registry 동작
		if request.Status == "STARTED" {
			if strings.HasPrefix(app.Entity.Name, "config") {
				configApp = app
				fmt.Printf("config_app: %s\n", configApp.Entity.Name)
			} else if strings.HasPrefix(app.Entity.Name, "registry") {
				// spring-cloud-config 동작 여부 확인
				basicUser := configApp.Entity.Environment[domain.BASIC_USER]
				basicSecret := configApp.Entity.Environment[domain.BASIC_SECRET]
				configAppName := configApp.Entity.Name

				rel, _ := regexp.Compile("[http]+s*://[a-zA-Z]{1,}[^\\.\\s]*[\\.]{1}")
				excludeString := rel.FindString(s.uaa.ApiURL)
				apiDomain := s.uaa.ApiURL[len(excludeString):]
				//apiDomain := strings.Trim(s.uaa.ApiURL, excludeString)
				fmt.Printf("apiDomain: %s\n", apiDomain)
				configApiUri := fmt.Sprintf("http://%s:%s@%s.%s/config/read?refresh=false",
					basicUser, basicSecret, configAppName, apiDomain)
				for i := 1; i <= maxCount; i++ {
					resp, err := http.Get(configApiUri)
					if err != nil {
						logger.Error("failed http get spring-cloud-config", err)
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(err)
						return
					}
					if resp.StatusCode != 200 {
						logger.Error(
							fmt.Sprintf("Config app is not running, StatusCode: %s",resp.StatusCode),
							fmt.Errorf("check cloudfoundry Config app"))
						if i == maxCount {
							w.WriteHeader(http.StatusInternalServerError)
							w.Write([]byte("Config Application is not working.\n Please try again in a moment."))
							return
						}
					} else {
						logger.Debug("Configuration application is running")
						break
					}
				}
			} else {
				logger.Error("[UpdateMicroserviceState] App does not exist. >>>", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		// cf application start/stop
		body := map[string]string{ "state" : request.Status, }
		_, err := s.UpdateApp(app.Meta.Guid, body, token)
		if err != nil {
			logger.Error("[UpdateMicroserviceState] UpdateApp err >>>", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// 1-2. gatewayapp
	gatewayapps, _ := s.GetAppByName(fmt.Sprintf("%s-%s", domain.MSA_GATEWAY_APP, request.Name))
	if len(gatewayapps.Resources) == 0 {
		http.Error(w, "err.Error()", http.StatusInternalServerError)
		return
	}

	gatewayapp := gatewayapps.Resources[0]
	body := map[string]string{ "state" : request.Status, }
	_, err = s.UpdateApp(gatewayapp.Meta.Guid, body, token)
	if err != nil {
		logger.Error("[UpdateMicroserviceState] UpdateApp err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. start/stop microservice apps
	msApps, err := s.repositoryFactory.Compose().ListMicroserviceAppApp(id)
	if err != nil {
		logger.Error("[UpdateMicroserviceState] failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	for _, app := range msApps {
		go func(appGuid string) {
			body := map[string]string{ "state" : request.Status, }
			_, err := s.UpdateApp(appGuid, body, token)
			if err != nil {
				logger.Error("[UpdateMicroserviceState] UpdateApp err >>>", err)
			}
		}(app.AppGuid)
	}

	// 3. UpdateMicroserviceStatus
	appRequest := domain.ComposeRequest{ID: id, Status: request.Status}
	_, err = s.repositoryFactory.Compose().UpdateMicroserviceStatus(appRequest)
	if err != nil {
		logger.Error("[UpdateMicroserviceState] UpdateMicroserviceStatus err >>>", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}