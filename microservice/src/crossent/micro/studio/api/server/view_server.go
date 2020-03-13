package server

import (
	"net/http"

	"encoding/json"
	"strconv"
	"fmt"
	"crossent/micro/studio/domain"
	"strings"
	"io/ioutil"
	"io"
	"encoding/xml"
	"github.com/cloudfoundry-community/go-cfclient"
	"code.cloudfoundry.org/bytefmt"
	"code.cloudfoundry.org/lager"
)


type route struct {
	ServiceName string `json:"serviceName"`
	Path string `json:"path"`
}



type instance struct {
	App string `xml:"app"`
	Ipaddr string `xml:"ipAddr"`
	Status string `xml:"status"`
}

type application struct {
	Name string `xml:"name"`
	Instance instance `xml:"instance"`
}

type applications struct {
	Application []application `xml:"application"`
}


type registry struct {
	ServiceName string `json:"serviceName"`
	AppName string `json:"appName"`
	Ip string `json:"ip"`
	Status string `json:"status"`
}




type property struct {
	AppName string `json:"appName"`
	Properties string `json:"properties"`
}

func (s *Server) ListMicroservice(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro")
	logger.Debug("ListMicroservice")

	offset := 0
	if ioffset, err := strconv.Atoi(r.FormValue("offset")); err == nil {
		offset = ioffset
	}
	name := r.URL.Query().Get("name")

	// 세션 사용자 아이디
	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	spaces, _ := s.userSpaces(userid)

	views, err := s.repositoryFactory.View().ListMicroservice(offset, name, spaces)


	if err != nil {
		logger.Error("failed ListMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	orgData := s.GetOrg()
	spaceData := s.GetSpace()
	serviceBindData := s.GetServiceBinding()


	for i, view := range views {
		// org name
		for _, value := range orgData.Resources {
			metadata := value.Metadata
			entity := value.Entity
			if metadata.GUID == view.OrgGuid {
				views[i].OrgName = entity.Name
			}
		}

		// space name
		for _, value := range spaceData.Resources {
			metadata := value.Metadata
			entity := value.Entity
			if metadata.GUID == view.SpaceGuid {
				views[i].SpaceName = entity.Name
			}
		}

		viewApps, err := s.repositoryFactory.View().ListMicroserviceAppApp(view.ID)

		if err != nil {
			logger.Error("failed ListMicroserviceAppApp", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}

		serviceCnt := 0
		serviceGuids := []string{}
		// App-Service 연결정보
		for _, viewApp := range viewApps {
			for _, bind := range serviceBindData.Resources {
				if bind.Entity.App_guid == viewApp.AppGuid {
					//serviceCnt += 1
					serviceGuids = append(serviceGuids, bind.Entity.Service_instance_guid)
				}
			}
		}
		// 중복제거
		keys := make(map[string]bool)
		for _, s := range serviceGuids {
			if _, value := keys[s]; !value {
				keys[s] = true
				serviceCnt += 1
			}
		}
		views[i].Service = serviceCnt
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(views)


}

func (s *Server) GetMicroservice(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro")
	logger.Debug("GetMicroservice")

	id := r.FormValue(":id")

	token, err := s.uaa.GetAuthToken()
	if err != nil {
		logger.Error("failed cf get auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	idint, _ := strconv.Atoi(id)
	view, err := s.repositoryFactory.View().GetMicroservice(idint)

	//add
	orgsummary := s.GetOrgSummary(view.OrgGuid, token)
	view.OrgName = orgsummary.Name

	spacesummary := s.GetSpaceSummary(view.SpaceGuid, token)
	view.SpaceName = spacesummary.Name


	if b := s.access(r, view.SpaceGuid); !b {
		logger.Error("no auth", fmt.Errorf("no auth"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no auth"))
		return
	}

	serviceBindData := s.GetServiceBinding()

	viewApps, err := s.repositoryFactory.View().ListMicroserviceAppApp(view.ID)

	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	serviceCnt := 0
	serviceGuids := []string{}
	//apps := []domain.View{}
	//services := []domain.View{}
	//servicesApps := []domain.View{}

	// App-Service 연결정보
	for _, viewApp := range viewApps {
		for _, bind := range serviceBindData.Resources {
			if bind.Entity.App_guid == viewApp.AppGuid {
				//serviceCnt += 1
				serviceGuids = append(serviceGuids, bind.Entity.Service_instance_guid)
			}
		}

		//summary := s.GetAppSummary(viewApp.AppGuid)
		//
		//for _, route := range summary.Routes {
		//	viewApp.Url = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
		//}
		//
		//viewApp.AppName = summary.Name
		//viewApp.App = summary.Instances
		//viewApp.Status = summary.State
		//
		//apps = append(apps, viewApp)
	}
	// 중복제거
	keys := make(map[string]bool)
	for _, s := range serviceGuids {
		if _, value := keys[s]; !value {
			keys[s] = true
			serviceCnt += 1
		}
	}
	view.Service = serviceCnt

	if err != nil {
		logger.Error("failed ListMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	type microserviceDetail struct {
		Microservice domain.View `json:"microservice"`
		Apps []domain.View `json:"apps"`
	}

	details := microserviceDetail{
		Microservice: view,
		//Apps: apps,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)


}

func (s *Server) GetMicroserviceDetail(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro")
	logger.Debug("GetMicroserviceDetail")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	view, err := s.repositoryFactory.View().GetMicroservice(idint)

	if err != nil {
		logger.Error("failed GetMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	if b := s.access(r, view.SpaceGuid); !b {
		logger.Error("no auth", fmt.Errorf("no auth"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no auth"))
		return
	}

	viewApps, err := s.repositoryFactory.View().ListMicroserviceAppApp(view.ID)

	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	// microservice 상세정보
	result := domain.MicroDetail{}

	token, err := s.uaa.GetAuthToken()
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

	apps := []domain.View{}
	services := []domain.View{}
	servicesApps := []domain.View{}
	policies := []domain.View{}

	routes := []route{}
	var routingData []byte

	registries := []registry{}
	var registryData []byte
	eurekaApps := applications{}

	properties := []property{}
	configUrl := ""

	for _, viewApp := range viewApps {
		summary := s.GetAppSummary(viewApp.AppGuid, token)
		if strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) {
			for _, route := range summary.Routes {
				configUrl = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
			}
			break
		}
	}

	for _, viewApp := range viewApps {

		// App 정보
		summary := s.GetAppSummary(viewApp.AppGuid, token)

		for _, route := range summary.Routes {
			viewApp.Url = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
		}

		viewApp.AppName = summary.Name
		viewApp.App = summary.Instances
		viewApp.Status = summary.State

		apps = append(apps, viewApp)

		for _, service := range summary.Services {
			serviceFind := 0

			viewApp.AppName = summary.Name
			viewApp.ServiceInstanceName = service.Name
			servicesApps = append(servicesApps, viewApp)

			for _, s := range services {
				if s.ServiceGuid == service.Guid {
					serviceFind += 1;
				}
			}
			if serviceFind == 0 {
				viewApp.AppName = summary.Name
				viewApp.ServiceName = service.Plan.Service.Label
				viewApp.ServiceInstanceName = service.Name
				viewApp.Plan = service.Plan.Name
				viewApp.ServiceGuid = service.Guid
				services = append(services, viewApp)
			}

		}

		// routing 정보
		if strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) {
			for _, route := range summary.Routes {
				viewApp.Url = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
				configUrl = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
			}
			//fmt.Println(viewApp.Url)
			basicUser := summary.Environment[domain.BASIC_USER]
			basicSecret := summary.Environment[domain.BASIC_SECRET]

			resp, err := http.Get(fmt.Sprintf("http://%s:%s@%s/config/read/apigateway?refresh=true", basicUser, basicSecret, viewApp.Url))
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

			if code := resp.StatusCode; code < 200 || code > 299 {
				routingData = nil
			} else {
				routingData = body
			}


		}

		// registry 정보
		if strings.HasPrefix(summary.Name, domain.MSA_REGISTRY_APP) {
			for _, route := range summary.Routes {
				viewApp.Url = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
			}
			//fmt.Println(viewApp.Url)
			basicUser := summary.Environment[domain.BASIC_USER]
			basicSecret := summary.Environment[domain.BASIC_SECRET]

			resp, err := http.Get(fmt.Sprintf("http://%s:%s@%s/eureka/apps", basicUser, basicSecret, viewApp.Url))
			if err != nil {
				logger.Error("failed http eureka", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if err != nil {
				logger.Error("failed http eureka response", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}

			if code := resp.StatusCode; code < 200 || code > 299 {
				registryData = nil
			} else {
				registryData = body
			}

		}


		// properties 정보
		if !strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) && configUrl != "" {
			for _, route := range summary.Routes {
				viewApp.Url = fmt.Sprintf("%s.%s", route.Host, route.Domain.Name)
			}

			basicUser := summary.Environment[domain.BASIC_USER]
			basicSecret := summary.Environment[domain.BASIC_SECRET]
			configUrl := fmt.Sprintf("http://%s:%s@%s/config/read/%s?refresh=true", basicUser, basicSecret, configUrl, summary.Name)
			resp, err := http.Get(configUrl)
			if err != nil {
				logger.Error("failed http read properties", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if err != nil {
				logger.Error("failed http read properties response", err)
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
						p.AppName = summary.Name
						p.Properties = s
						properties = append(properties, p)
					}
				}
			}

		}
	}

	// cf-networking 정보
	access, _ := s.GetAccess()
	for _, policy := range access.Policies {
		sourceFind := false
		targetFind := false
		for _, viewApp := range viewApps {
			if viewApp.AppGuid == policy.Source.ID {
				sourceFind = true
			}
			if viewApp.AppGuid == policy.Destination.ID {
				targetFind = true
			}
		}
		if sourceFind || targetFind {
			v := domain.View{}
			for _, a := range apps {
				if a.AppGuid == policy.Source.ID {
					v.Source = a.AppName
					if app, err := cf.GetAppByGuid(policy.Destination.ID); err == nil {
						v.Target = app.Name
					}
				}
				if a.AppGuid == policy.Destination.ID {
					v.Target = a.AppName
					if app, err := cf.GetAppByGuid(policy.Source.ID); err == nil {
						v.Source = app.Name
					}
				}
			}
			v.Port = policy.Destination.Port.Start
			policies = append(policies, v)
		}
	}

	result.App = apps
	result.Service = services
	result.Policy = policies
	result.ServiceApp = servicesApps

	// routing 정보
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

	// registry 정보
	if registryData != nil {
		err := xml.Unmarshal(registryData, &eurekaApps)
		if err != nil {
			logger.Error("failed registry response xml unmarshal", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}

		for _, val := range eurekaApps.Application {
			r := registry{}
			r.ServiceName = val.Name
			r.AppName = val.Instance.App
			r.Ip = val.Instance.Ipaddr
			r.Status = val.Instance.Status
			registries = append(registries, r)
		}
	}


	type microserviceDetail struct {
		Microservice domain.View `json:"microservice"`
		Apps domain.MicroDetail `json:"apps"`
		Routes []route `json:"routes"`
		Registries []registry `json:"registries"`
		Properties []property `json:"properties"`
	}

	details := microserviceDetail{
		Apps: result,
		Routes: routes,
		Registries: registries,
		Properties: properties,
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)


}


func (s *Server) GetMicroserviceLink(w http.ResponseWriter, r *http.Request) {

	logger := s.logger.Session("micro")
	logger.Debug("GetMicroserviceLink")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	view, err := s.repositoryFactory.View().GetMicroservice(idint)

	if err != nil {
		logger.Error("failed GetMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	if b := s.access(r, view.SpaceGuid); !b {
		logger.Error("no auth", fmt.Errorf("no auth"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no auth"))
		return
	}

	viewApps, err := s.repositoryFactory.View().ListMicroserviceAppApp(view.ID)

	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	// external api
	viewApis, err := s.repositoryFactory.Apigateway().ListMicroserviceAppApiByMicroID(view.ID)
	if err != nil {
		logger.Error("failed ListMicroserviceAppApiByMicroID", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	token, err := s.uaa.GetAuthToken()
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

	type node struct {
		ID string `json:"id"`
		Type string `json:"type"`
		Name string `json:"name"`
		Active string `json:"active"`
		Essential string `json:"essential"`
		Group int `json:"group"`
		Cpu string `json:"cpu"`
		Memory string `json:"memory"`
		Disk string `json:"disk"`
	}

	type link struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type string `json:"type"`
		Group int `json:"group"`
	}


	nodes := []node{}
	links := []link{}
	services := []node{}

	nodes_services := []node{} // node 형태의 서비스 정보

	frontend_app_guid := ""


	for _, viewApp := range viewApps {
		// App summary 정보
		summary := s.GetAppSummary(viewApp.AppGuid, token)

		var c, m, d string

		if strings.ToUpper(summary.State) == "STARTED" {
			// App 상태 정보
			stats, err := cf.GetAppStats(viewApp.AppGuid)
			if err != nil {
				logger.Error("failed cf GetAppStats", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}
			c, m, d = appUsage(stats)
		}

		if !strings.HasPrefix(summary.Name, domain.MSA_CONFIG_APP) && !strings.HasPrefix(summary.Name, domain.MSA_REGISTRY_APP) {

			n := node{viewApp.AppGuid, "App", summary.Name, summary.State, viewApp.Essential, 0, c, m, d}
			nodes = append(nodes, n)
		}
		//else {
		//	n := node{viewApp.AppGuid, "App", summary.Name, summary.State, 0, c, m, d}
		//	nodes_services = append(nodes_services, n)
		//}

		if viewApp.Essential == "front" {
			frontend_app_guid = viewApp.AppGuid
		}

		// Service 정보
		for _, service := range summary.Services {
			serviceFind := 0

			if !strings.HasPrefix(service.Name, domain.SERVICE_CONFIG_SERVER) && !strings.HasPrefix(service.Name, domain.SERVICE_REGISTRY_SERVER) {
				// App-Service 연결정보
				l := link{viewApp.AppGuid, service.Guid, "Service", 9}
				links = append(links, l)

				for _, s := range services {
					if s.ID == service.Guid {
						serviceFind += 1;
					}
				}
				if serviceFind == 0 {
					n := node{service.Guid, "Service", service.Name, service.Plan.Name, "", 9, "", "", ""}
					services = append(services, n)
				}
			}

		}
	}

	// external api
	for _, viewApi := range viewApis {
		n := node{strconv.Itoa(viewApi.ID), "API", viewApi.Name, viewApi.Path, "", 12, "", "", ""}
		nodes = append(nodes, n)
		// frontend - api
		if frontend_app_guid != "" {
			l := link{frontend_app_guid, strconv.Itoa(viewApi.ID), "App", 12}
			links = append(links, l)
		}
	}

	// node_service
	for _, ns := range nodes_services {
		for i, s := range services {
			if strings.Contains(ns.Name, "config") && strings.Contains(s.Name, "config") {
				services[i].Cpu = ns.Cpu
				services[i].Memory = ns.Memory
				services[i].Disk = ns.Disk
			}
			if strings.Contains(ns.Name, "registry") && strings.Contains(s.Name, "registry") {
				services[i].Cpu = ns.Cpu
				services[i].Memory = ns.Memory
				services[i].Disk = ns.Disk
			}
		}
	}

	// cf-networking 정보
	access, err := s.GetAccess()
	if err == nil {
		for _, policy := range access.Policies {
			sourceFind := false
			targetFind := false
			for _, viewApp := range viewApps {
				if viewApp.AppGuid == policy.Source.ID {
					sourceFind = true
				}
				if viewApp.AppGuid == policy.Destination.ID {
					targetFind = true
				}
			}
			if sourceFind && targetFind {
				l := link{}
				for _, a := range nodes {
					if a.ID == policy.Source.ID {
						l.Source = policy.Source.ID
						l.Target = policy.Destination.ID
					}
					if a.ID == policy.Destination.ID {
						l.Source = policy.Source.ID
						l.Target = policy.Destination.ID
					}
				}
				l.Group = 0
				l.Type = "App"
				links = append(links, l)
			}
		}
	} else {
		logger.Error("failed policy", err)
	}


	nodes = append(nodes, services...)


	type result struct {
		Nodes []node `json:"nodes"`
		Links []link `json:"links"`
	}

	rtn := result{Nodes: nodes, Links: links}

	//fmt.Println(l)

	//b := new(bytes.Buffer)
	//json.NewEncoder(b).Encode(ll)
	//
	//fmt.Println(b)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rtn); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}


}


//func (s *Server) ListMicroserviceApi(w http.ResponseWriter, r *http.Request) {
//	logger := s.logger.Session("micro")
//	logger.Debug("ListMicroserviceApi")
//
//	offset := 0
//	if ioffset, err := strconv.Atoi(r.FormValue("offset")); err == nil {
//		offset = ioffset
//	}
//	name := r.URL.Query().Get("name")
//
//	// 세션 사용자 아이디
//	session := domain.SessionManager.Load(r)
//	userid, _ := session.GetString(domain.USER_ID)
//
//	// 세션 사용자 속한 모든 space_guid
//	spaces, _ := s.userSpaces(userid)
//
//	views, err := s.repositoryFactory.View().ListMicroserviceApi(offset, name, spaces)
//
//
//	if err != nil {
//		logger.Error("failed ListMicroserviceApi", err)
//		w.WriteHeader(http.StatusInternalServerError)
//		json.NewEncoder(w).Encode(err)
//		return
//	}
//
//	w.WriteHeader(http.StatusOK)
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(views)
//
//
//}
//
//func (s *Server) SaveMicroserviceApi(w http.ResponseWriter, r *http.Request) {
//	logger := s.logger.Session("micro")
//	logger.Debug("SaveMicroserviceApi")
//
//	// 세션 사용자 아이디
//	session := domain.SessionManager.Load(r)
//	userid, _ := session.GetString(domain.USER_ID)
//
//	// 세션 사용자 속한 모든 space_guid
//	spaces, _ := s.userSpaces(userid)
//
//	id := r.FormValue(":id")
//
//	idint, _ := strconv.Atoi(id)
//
//	var view domain.View
//	err := json.NewDecoder(r.Body).Decode(&view)
//	if err != nil {
//		logger.Error("Decode err >>>", err)
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	resp, err := http.Get(fmt.Sprintf("http://%s/v2/api-docs", view.Url))
//	if err != nil {
//		logger.Error("failed http get api-docs", err)
//		w.WriteHeader(http.StatusInternalServerError)
//		json.NewEncoder(w).Encode(err)
//		return
//	}
//	defer resp.Body.Close()
//	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
//	if err != nil {
//		logger.Error("failed http get api-docs response", err)
//		w.WriteHeader(http.StatusInternalServerError)
//		json.NewEncoder(w).Encode(err)
//		return
//	}
//
//	if code := resp.StatusCode; code < 200 || code > 299 {
//		w.WriteHeader(http.StatusInternalServerError)
//		json.NewEncoder(w).Encode(resp.StatusCode)
//		return
//	} else {
//
//		view.ID = idint
//		view.Swagger = string(body)
//		err = s.repositoryFactory.View().SaveMicroserviceApi(view, spaces)
//
//		if err != nil {
//			logger.Error("failed SaveMicroserviceApi", err)
//			w.WriteHeader(http.StatusInternalServerError)
//			json.NewEncoder(w).Encode(err)
//			return
//		}
//	}
//
//
//	w.WriteHeader(http.StatusOK)
//	w.Header().Set("Content-Type", "application/json")
//
//
//}
//
//func (s *Server) GetMicroserviceApi(w http.ResponseWriter, r *http.Request) {
//	logger := s.logger.Session("micro")
//	logger.Debug("GetMicroserviceApi")
//
//	id := r.FormValue(":id")
//
//	idint, _ := strconv.Atoi(id)
//	view, err := s.repositoryFactory.View().GetMicroservice(idint)
//
//	if b := s.access(r, view.SpaceGuid); !b {
//		logger.Error("no auth", fmt.Errorf("no auth"))
//		w.WriteHeader(http.StatusInternalServerError)
//		w.Write([]byte("no auth"))
//		return
//	}
//
//	if err != nil {
//		logger.Error("failed GetMicroservice", err)
//		w.WriteHeader(http.StatusInternalServerError)
//		json.NewEncoder(w).Encode(err)
//		return
//	}
//
//
//	w.WriteHeader(http.StatusOK)
//	w.Header().Set("Content-Type", "application/json")
//	//json.NewEncoder(w).Encode(view.Swagger)
//	w.Write([]byte(view.Swagger))
//
//
//}

func (s *Server) DeleteMicroservice(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro")
	logger.Debug("DeleteMicroservice")

	id := r.FormValue(":id")
	idint, _ := strconv.Atoi(id)

	microApis, err := s.repositoryFactory.Apigateway().ListMicroserviceAppApiByMicroID(idint)
	if err != nil {
		logger.Error("failed MicroAppAPI by MicroID", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	} else if len(microApis) > 0 {
		logger.Error("MicroAppAPI exists, First, delete MicroAppAPI !", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("There is a registered API int this MSA, First, delete API !"))
		return
	} else {

	}

	view, err := s.repositoryFactory.View().GetMicroservice(idint)

	adminToken, err := s.uaa.GetAuthToken()
	if err != nil {
		logger.Error("failed cf get auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	if err != nil {
		logger.Error("failed GetMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	if b := s.access(r, view.SpaceGuid); !b {
		logger.Error("no auth", fmt.Errorf("no auth"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no auth"))
		return
	}

	if view.Visible == "public" {
		logger.Error("public", fmt.Errorf("public"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("public microservice cannot delete"))
		return
	}


	viewApps, err := s.repositoryFactory.View().ListMicroserviceAppApp(view.ID)

	if err != nil {
		logger.Error("failed ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	viewServices, err := s.repositoryFactory.View().ListMicroserviceAppService(view.ID)

	if err != nil {
		logger.Error("failed ListMicroserviceAppService", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	for _, viewService := range viewServices {
		if apps, err := s.GetAppByName(fmt.Sprintf("%s%s", domain.MSA_CONFIG_APP, viewService.ServiceGuid)); err == nil {
			if len(apps.Resources) > 0 {
				app := apps.Resources[0]
				view := domain.View{AppGuid: app.Meta.Guid}
				viewApps = append(viewApps, view)
			}
		}
		if apps, err := s.GetAppByName(fmt.Sprintf("%s%s", domain.MSA_REGISTRY_APP, viewService.ServiceGuid)); err == nil {
			if len(apps.Resources) > 0 {
				app := apps.Resources[0]
				view := domain.View{AppGuid: app.Meta.Guid}
				viewApps = append(viewApps, view)
			}
		}
	}

	cf, err := s.uaa.CfClient()
	if err != nil {
		logger.Error("failed cf client", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	msApps, err := s.repositoryFactory.Compose().ListMicroserviceAppApp(view.ID)
	// micro_app 테이블에서 레코드를 통째로 삭제할지 말지 확인
	deleteMicroAppRecord := false
	var appGuid *string = nil
	// viewApps 중 같은 마이크로 서비스인 경우 삭제
	if err != nil {
		logger.Error("failed Compose().ListMicroserviceAppApp", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	} else {
		if msApps == nil {
			deleteMicroAppRecord = true
		} else {
			for _, msApp := range msApps {
				fmt.Println("will Delete msApp_ID: "+ string(msApp.MicroID) + " msApp_NAME: " + view.Name)
				appGuid = &msApp.AppGuid
				if appGuid == nil || *appGuid == ""  {
					deleteMicroAppRecord = true
					break
				}
			}
		}
	}

	err = s.repositoryFactory.View().DeleteMicroservice(view.ID, view.Name, deleteMicroAppRecord)
	if err != nil {
		logger.Error("failed DeleteMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	sharedDomains, err := s.ListSpaceDomains(view.SpaceGuid)
	if err != nil {
		logger.Error("ListSpaceDomains err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sharedDomain := sharedDomains.Resources[0]

	for _, viewApp := range viewApps {
		go func(appGuid string) {
			// app stop
			body := map[string]string{ "state" : domain.APP_STATE_STOPPED, }
			_, err = s.UpdateApp(appGuid, body, adminToken)
			if err != nil {
				logger.Error("DeleteMicroservice UpdateApp-Stop err >>>", err)
			}

			if sb, err := s.ListServiceBindingByApp(appGuid); err == nil {
				for _, sbr := range sb.Resources {
					// delete service binding
					//if err := s.DeleteServiceBindingByApp(appGuid, sbr.Meta.Guid); err != nil {
					//	logger.Error("DeleteMicroservice Servicebindingg error", err, lager.Data{"appGUID": appGuid})
					//}
					// delete service binding
					//if err := s.DeleteServiceBinding(sbr.Entity.ServiceInstanceGuid); err != nil {
					//	logger.Error("DeleteMicroservice DeleteServiceBinding error", err, lager.Data{"appGUID": sbr.Entity.ServiceInstanceGuid})
					//}

					// delete service instance (recursive)
					if err := s.DeleteServiceInstanceByGuid(sbr.Entity.ServiceInstanceGuid); err != nil {
						logger.Error("DeleteMicroservice Serviceinstance error", err, lager.Data{"appGUID": appGuid})
					}
				}
			} else {
				//allServiceInstances, err := s.ListServiceInstance()
				for _, viewService := range viewServices {
					service := s.GetServiceInstanceByGuid (viewService.ServiceGuid)
					if &service != nil {
						if err := s.DeleteServiceInstanceByGuid(service.Metadata.GUID); err != nil {
							logger.Error("DeleteMicroservice Serviceinstance error", err, lager.Data{"appGUID": appGuid})
						}
					}
				}
			}



			// delete route
			if listroutes, err := s.ListRouteForApp(appGuid); err == nil {
				for _, route := range listroutes.Resources {
					if err := s.DeleteRoute(route.Meta.Guid); err != nil {
						logger.Error("DeleteRouteForApp error", err, lager.Data{"appGUID": appGuid})
					}
				}
			}


			// delete app
			if err := cf.DeleteApp(appGuid); err != nil {
				logger.Error("DeleteMicroservice error", err, lager.Data{"appGUID": appGuid})
			}

		}(viewApp.AppGuid)

		// monitoring
		if viewApp.Essential == string(domain.MsMonitoring) {
			//go func(appGuid string) {
			//var datasourceId struct{id string}

			spaceName, err := s.userSpaceName(view.SpaceGuid)
			if err != nil {
				logger.Error("failed userSpaceName", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}

			if err = requestGrafanaServer("DELETE", fmt.Sprintf("http://%s.%s", fmt.Sprintf("%s-dashboard-%s", view.Name, spaceName), sharedDomain.Entity.Name), s.uaa.GrafanaAdminPassword, fmt.Sprintf("/api/datasources/name/%s_datasource", view.Name), nil, nil, logger); err != nil {
				logger.Error("http delete grafana datasources request", err)
				//http.Error(w, err.Error(), http.StatusInternalServerError)
				//return
			}

			if err = requestGrafanaServer("DELETE", fmt.Sprintf("http://%s.%s", fmt.Sprintf("%s-dashboard-%s", view.Name, spaceName), sharedDomain.Entity.Name), s.uaa.GrafanaAdminPassword, fmt.Sprintf("/api/dashboards/uid/micrometer-%s", view.Name), nil, nil, logger); err != nil {
				logger.Error("http delete grafana dashboards request", err)
				//http.Error(w, err.Error(), http.StatusInternalServerError)
				//return
			}

			//}(viewApp.AppGuid)
		}
	}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

}

func appUsage(appstats map[string]cfclient.AppStats) (string, string, string) {
	if app, exists := appstats["0"]; exists {
		dqs := bytefmt.ByteSize(uint64(app.Stats.DiskQuota))
		mqs := bytefmt.ByteSize(uint64(app.Stats.MemQuota))
		ds := bytefmt.ByteSize(uint64(app.Stats.Usage.Disk))
		ms := bytefmt.ByteSize(uint64(app.Stats.Usage.Mem))
		c := app.Stats.Usage.CPU * 100

		return fmt.Sprintf("cpu: %.1f %%", c),
			fmt.Sprintf("memory: %s/%s", ms, mqs),
			fmt.Sprintf("disk: %s/%s", ds, dqs)
	}

	return "", "", ""

}

// 접근 권한 체크
func (s *Server) access(r *http.Request, spaceGuid string) bool {
	// 세션 사용자 아이디
	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	spaces, _ := s.userSpaces(userid)

	return contains(spaces, spaceGuid)
}

func contains(arr []string, str string) bool {
	//fmt.Println(">>>>", arr)
	//fmt.Println(">>",str)
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}


func (s *Server) userOrgName(orgguid string) (string, error) {
	logger := s.logger.Session("micro_server")
	logger.Debug("userOrgName")


	cf, err := s.uaa.CfClient()
	if err != nil {
		logger.Error("failed cf client", err)
		return "", err
	}

	// space 권한 조회
	org, err := cf.GetOrgByGuid(orgguid)
	if err != nil {
		logger.Error("failed cf GetOrgByGuid", err)
		return "", err
	}

	return org.Name, nil
}

func (s *Server) userSpaceName(spaceguid string) (string, error) {
	logger := s.logger.Session("micro_server")
	logger.Debug("userSpaceName")


	cf, err := s.uaa.CfClient()
	if err != nil {
		logger.Error("failed cf client", err)
		return "", err
	}

	// space 권한 조회
	space, err := cf.GetSpaceByGuid(spaceguid)
	if err != nil {
		logger.Error("failed cf GetSpaceByGuid", err)
		return "", err
	}

	return space.Name, nil
}