package server

import (
	"net/http"

	"encoding/json"
	"fmt"
	"strconv"
	"crossent/micro/studio/client"
	"crossent/micro/studio/domain"
	"net/url"
	"github.com/cloudfoundry-community/go-cfclient"
	"encoding/base64"
)

type OrgResponse struct {
	Resources []struct {
		Metadata struct {
				 GUID	string `json:"guid"`
			 } `json:"metadata"`
		Entity struct {
				 Name             string `json:"name"`
				 Status string `json:"status"`
			 } `json:"entity"`
	} `json:"resources"`
}

type SpaceResponse struct {
	Resources []struct {
		Metadata struct {
				 GUID	string `json:"guid"`
			 } `json:"metadata"`
		Entity struct {
				 Name             string `json:"name"`
				 Organization_guid string `json:"organization_guid"`
			 } `json:"entity"`
	} `json:"resources"`
}

type AppResponse struct {
	Total_results int `json:"total_results"`
	Total_pages int    `json:"total_pages"`
	Resources []struct {
		Metadata struct {
				 GUID	string `json:"guid"`
			 } `json:"metadata"`
		Entity struct {
				 Name             string `json:"name"`
				 Memory int32 `json:"memory"`
				 Disk_quota int32 `json:"disk_quota"`
				 Buildpack string `json:"buildpack"`
				 State string `json:"state"`
				 Space_guid string `json:"space_guid"`
				 Instances int32 `json:"instances"`
				 Environment map[string]string `json:"environment_json"`
			 } `json:"entity"`
	} `json:"resources"`
}

type OrgSummary struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
	Status string `json:"status"`
	Spaces []struct{
		Guid string `json:"guid"`
		Name string `json:"name"`
		Service_count int32 `json:"service_count"`
		App_count int32 `json:"app_count"`
		Mem_dev_total int32 `json:"mem_dev_total"`
		Mem_prod_total int32 `json:"mem_prod_total"`
	}
}

type SpaceSummary struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
	Status string `json:"status"`
	Apps []apps `json:"apps"`
	Services []services `json:"services"`
}

type AppSummary struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
	Routes []struct{
		Host string `json:"host"`
		Domain struct {
			Name string `json:"name"`
		       } `json:"domain"`
	}
	Services []struct{
		Guid string `json:"guid"`
		Name string `json:"name"`
		Plan struct {
			Name string `json:"name"`
			Service struct {
					Label string `json:"label"`
				} `json:"service"`
		     } `json:"service_plan"`
	}
	Instances int `json:"instances"`
	Memory int    `json:"memory"`
	DiskQuota int `json:"disk_quota"`
	State string `json:"state"`
	Buildpack string `json:"buildpack"`
	Environment map[string]string `json:"environment_json"`
}

type Policies struct {
	Source      struct {
			    ID  string `json:"id"`
		    }      `json:"source"`
	Destination struct {
			    ID       string `json:"id"`
			    Port struct {
					  Start int `json:"start"`
					  End   int `json:"end"`
				  } `json:"ports"`
			    Protocol string `json:"protocol"`
		    } `json:"destination"`
}

type AccessResponse struct {
	Policies []Policies `json:"policies"`
}

func (s *Server) ListOrg(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("ListOrg")

	route := fmt.Sprintf("/v2/organizations")
	var response OrgResponse
	//error := s.uaa.GetResources("GET", route, nil, &response)
	session := domain.SessionManager.Load(r)
	access_token, err := session.GetString(domain.UAA_TOKEN_NAME)
	if err != nil {
		s.logger.Error("[ListOrg] failed cf get auth token", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	err = s.uaa.GetResourcesFromToken("GET", route, nil, &response, access_token)
	if err != nil {
		logger.Error("ListOrg", err)
		var cfErrBody domain.CloudFoundryErrBody
		json.Unmarshal([]byte(err.Error()), &cfErrBody)
		var code int
		if cfErrBody.CfErrorCode == "CF-InvalidAuthToken" {
			code = cfErrBody.HttpStatusCode
		} else {
			code = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("ListOrg", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
	}
}

type links struct {
	SourceId string `json:"source_id"`
	TargetId string `json:"target_id"`
	Status string `json:"status"`
}

type tops struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`

}
type orgs struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`

}
type spaces struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`

}
type apps struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`

}
type services struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Status string `json:"status"`

}

type LinkNode struct {

	Links []links `json:"links"`
	Tops []tops `json:"tops"`
	Orgs []orgs `json:"orgs"`
	Spaces []spaces `json:"spaces"`
	Apps []apps `json:"apps"`
	Services []services `json:"services"`
}

type wireInfo struct{
	ID string `json:"target_id"`
	Protocol string `json:"protocol"`
	Port int `json:"port"`
	Bind_id string `json:"bind_id"`
}


type connNode struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Active string `json:"active"`
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
	Wires [][]string `json:"wires"`
	Org_id string `json:"orgid"`
	Space_id string `json:"spaceid"`
	WiresInfo []wireInfo `json:"wires_info"`
}

type ServiceResource struct {
	Metadata struct {
			 GUID	string `json:"guid"`
		 } `json:"metadata"`
	Entity struct {
			 Name             string `json:"name,omitempty"`
			 Service_guid string `json:"service_guid,omitempty"`
			 Service_plan_guid string `json:"service_plan_guid,omitempty"`
			 Space_guid string `json:"space_guid,omitempty"`
			 Service_instance_guid string `json:"service_instance_guid,omitempty"`
			 App_guid string `json:"app_guid,omitempty"`
			 Type string `json:"type,omitempty"`
			 Label string `json:"label,omitempty"`
			 Tags []string `json:"tags,omitempty"`
		 } `json:"entity"`
}

type ServiceResponse struct {
	Resources []ServiceResource `json:"resources"`
}

type BindingService struct {
	Service_instance_guid string `json:"service_instance_guid"`
	App_guid string `json:"app_guid"`
}

func (s *Server) GetOrg () OrgResponse {

	route := fmt.Sprintf("/v2/organizations")
	var response OrgResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return  response
}

func (s *Server) GetOrgSummary (orgguid string, token *client.CF_TOKEN) OrgSummary {

	summary := fmt.Sprintf("/v2/organizations/%v/summary", orgguid)
	var response OrgSummary
	err := s.uaa.GetResourcesFromToken("GET", summary, nil, &response, token.AccessToken)

	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return response
}

func (s *Server) GetSpaceSummary (spaceguid string, token *client.CF_TOKEN) SpaceSummary {

	summary := fmt.Sprintf("/v2/spaces/%v/summary", spaceguid)
	var response SpaceSummary
	err := s.uaa.GetResourcesFromToken("GET", summary, nil, &response, token.AccessToken)

	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return response
}


func (s *Server) ListSpace (w http.ResponseWriter, r *http.Request) {

	route := fmt.Sprintf("/v2/spaces")
	var response SpaceResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (s *Server) ListOrgSpace (w http.ResponseWriter, r *http.Request) {

	orgid := r.FormValue(":orgid")
	route := fmt.Sprintf("/v2/organizations/%v/spaces", orgid)
	var response SpaceResponse
	//error := s.uaa.GetResources("GET", route, nil, &response)
	session := domain.SessionManager.Load(r)
	access_token, err := session.GetString(domain.UAA_TOKEN_NAME)
	if err != nil {
		s.logger.Error("[ListOrgSpace] failed cf get auth token", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	error := s.uaa.GetResourcesFromToken("GET", route, nil, &response, access_token)

	if error != nil {
		//panic(error)
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (s *Server ) ListManageOrg(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro_server")
	logger.Debug("ListManageOrg")

	route := fmt.Sprintf("/v2/organizations")
	var response OrgResponse
	//error := s.uaa.GetResources("GET", route, nil, &response)
	session := domain.SessionManager.Load(r)
	access_token, err := session.GetString(domain.UAA_TOKEN_NAME)
	if err != nil {
		s.logger.Error("[ListOrg] failed cf get auth token", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	sessionUserid, _ := session.GetString(domain.USER_ID)

	cf, err := s.uaa.CfClient()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	err = s.uaa.GetResourcesFromToken("GET", route, nil, &response, access_token)
	if err != nil {
		logger.Error("ListOrg", err)
		var cfErrBody domain.CloudFoundryErrBody
		json.Unmarshal([]byte(err.Error()), &cfErrBody)
		var code int
		if cfErrBody.CfErrorCode == "CF-InvalidAuthToken" {
			code = cfErrBody.HttpStatusCode
		} else {
			code = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), code)
		return
	}

	result := OrgResponse{}

	for _, org := range response.Resources {
		if user, err := cf.ListOrgManagers(org.Metadata.GUID); err == nil {
			for _, u := range user {
				if u.Username == sessionUserid {
					result.Resources = append(result.Resources, org)
				}
			}
		}

	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(result); err != nil {
		logger.Error("ListOrg", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
	}
}

func (s *Server) GetSpace () SpaceResponse{

	route := fmt.Sprintf("/v2/spaces")
	var response SpaceResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return response
}

func (s *Server) ListApp (w http.ResponseWriter, r *http.Request) {

	route := fmt.Sprintf("/v2/apps")
	q := r.FormValue("q")
	if q != "" {
		route = fmt.Sprintf("%v?q=%v", route, q)
	}

	var response AppResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}

	//fmt.Fprintln(resp, response)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (s *Server) ListAppByEnv (w http.ResponseWriter, r *http.Request) {
	 // session user id
	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	name := r.FormValue("name")
	envName := r.FormValue("env")
	envPrivate := r.FormValue("private")

	var result AppResponse
	var response AppResponse

	// loop를 돌리기 위한 첫 total_page값
	result.Total_pages = 2
	for currentPage := 1; currentPage <= result.Total_pages; currentPage++ {
		route := fmt.Sprintf("/v2/apps?results-per-page=%v", 50)
		if name != "" {
			route = fmt.Sprintf("%v?q=name:%v", route, name)
		} else {
			//route = fmt.Sprintf("%v?results-per-page=%v", route, 10)
			route = fmt.Sprintf("%v&page=%v", route, currentPage)
		}

		error := s.uaa.GetResources("GET", route, nil, &result)
		if error != nil {
			http.Error(w, error.Error(), http.StatusInternalServerError)
			return
		}

		for _, app := range result.Resources {
			if _, ok := app.Entity.Environment[envName]; ok {
				if _, private := app.Entity.Environment[envPrivate]; private {
					if app.Entity.Environment[envPrivate] == "" || userid == app.Entity.Environment[envPrivate] {
						response.Resources = append(response.Resources, app)
					}
				}else{
					response.Resources = append(response.Resources, app)
				}
			}
		}
	}
	response.Total_pages = result.Total_pages
	response.Total_results = result.Total_results

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) GetApp () AppResponse {

	route := fmt.Sprintf("/v2/apps")
	var response AppResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return response
}

func (s *Server) ListAccess (w http.ResponseWriter, r *http.Request) {

	route := fmt.Sprintf("/networking/v1/external/policies")
	var response AccessResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}

	//fmt.Fprintln(resp, response)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (s *Server) GetAccess () (AccessResponse, error) {

	route := fmt.Sprintf("/networking/v1/external/policies")
	var response AccessResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//fmt.Println(error)
		s.logger.Error("GetAccess", error)
	}

	return response, error
}

func (s *Server) GetAccessById (id string) (AccessResponse, error) {
	route := fmt.Sprintf("/networking/v1/external/policies?id=%s", id)
	var response AccessResponse
	err := s.uaa.GetResources("GET", route, nil, &response)
	if err != nil {
		s.logger.Error("GetAccess", err)
	}
	return response, err
}

func (s *Server) CreateAccess (reqData interface{}) error {

	token, err := s.uaa.GetAuthToken()
	if err != nil {
		s.logger.Error("[CreateAccess] failed cf get auth token", err)
		return err
	}

	route := fmt.Sprintf("/networking/v1/external/policies")
	var response AccessResponse
	err = s.uaa.GetResourcesFromToken("POST", route, reqData, &response, token.AccessToken)
	if err != nil {
		//panic(error)
		s.logger.Error(err.Error(), err)
	}

	return err
}

func (s *Server) DeleteAccess (reqData interface{}) error {

	route := fmt.Sprintf("/networking/v1/external/policies/delete")
	var response AccessResponse
	error := s.uaa.GetResources("POST", route, reqData, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return error
}

func (s *Server) ListLink(w http.ResponseWriter, r *http.Request) {

	orgData := s.GetOrg()
	spaceData := s.GetSpace()
	appData := s.GetApp()
	serviceData := s.GetServiceInstance()


	l := LinkNode{}
	t := tops{"1234567890", " ", "Active"}
	l.Tops = append(l.Tops, t)
	//l.Routers = append(l.Routers, routers{})
	//l.Networks = append(l.Networks, networks{})

	for _, value := range orgData.Resources {
		metadata := value.Metadata
		entity := value.Entity

		o := orgs{metadata.GUID, entity.Name, entity.Status}
		l.Orgs = append(l.Orgs, o)
		lk := links{"1234567890", metadata.GUID, "Active"}
		l.Links = append(l.Links, lk)
		for _, value2 := range spaceData.Resources {
			metadata2 := value2.Metadata
			entity2 := value2.Entity


			if metadata.GUID == entity2.Organization_guid {
				s := spaces{metadata2.GUID, entity2.Name, "Active"}
				l.Spaces = append(l.Spaces, s)
				lk := links{metadata.GUID, metadata2.GUID, "Active"}
				l.Links = append(l.Links, lk)
				for _, value3 := range appData.Resources {
					metadata3 := value3.Metadata
					entity3 := value3.Entity


					if metadata2.GUID == entity3.Space_guid {
						a := apps{metadata3.GUID, entity3.Name, entity3.State}
						l.Apps = append(l.Apps, a)
						lk := links{metadata2.GUID, metadata3.GUID, "Active"}
						l.Links = append(l.Links, lk)
					}
				}

				for _, value4 := range serviceData.Resources {
					metadata4 := value4.Metadata
					entity4 := value4.Entity
					if metadata2.GUID == entity4.Space_guid {
						sv := services{metadata4.GUID, entity4.Name, "Active"}
						l.Services = append(l.Services, sv)
					}
				}
			}
		}
	}

	// App 연결정보 설정
	accessData, error := s.GetAccess()
	if error == nil {
		for _, acc := range accessData.Policies {
			lk := links{acc.Source.ID, acc.Destination.ID, "Access"}
			l.Links = append(l.Links, lk)
		}
	}

	// App-Service 연결정보 설정
	serviceBindData := s.GetServiceBinding()
	for _, bind := range serviceBindData.Resources {
		lk := links{bind.Entity.App_guid, bind.Entity.Service_instance_guid, "Active"}
		l.Links = append(l.Links, lk)
	}

	fmt.Println(l)

	//b := new(bytes.Buffer)
	//json.NewEncoder(b).Encode(l)
	//
	//fmt.Println(b)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(l); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (s *Server) ListConnect(w http.ResponseWriter, r *http.Request) {

	orgData := s.GetOrg()
	spaceData := s.GetSpace()
	appDaa := s.GetApp()

	//fmt.Println(spaceid)

	l := []connNode{}

	iX := 50 //192
	iY := 150 //172
	iZ := 65535
	org := []string{}

	for _, value := range orgData.Resources {
		metadata := value.Metadata
		entity := value.Entity

		o := connNode{metadata.GUID, "org", entity.Name, entity.Status, iX, iY, iZ, nil, metadata.GUID, "", nil}
		l = append(l, o)

		for i, value2 := range spaceData.Resources {
			metadata2 := value2.Metadata
			entity2 := value2.Entity


			if metadata.GUID == entity2.Organization_guid {
				org = append(org, metadata.GUID)
				s := connNode{metadata2.GUID, "space", entity2.Name, "Active", iX+(i+150), iY, iZ, nil, metadata.GUID, metadata2.GUID, nil}
				l = append(l, s)

				for i, v := range l {
					if v.Org_id == metadata.GUID && v.Type == "org" {
						if len(l[i].Wires) == 0 {
							l[i].Wires = append(l[i].Wires, []string{metadata2.GUID})
						}else{
							l[i].Wires[0] = append(l[i].Wires[0], metadata2.GUID)
						}
					}
				}

				for j, value3 := range appDaa.Resources {
					metadata3 := value3.Metadata
					entity3 := value3.Entity


					if metadata2.GUID == entity3.Space_guid {
						a := connNode{metadata3.GUID, "app", entity3.Name, entity3.State, iX+(i+1+150)+300, iY+(j*50), iZ, nil, metadata.GUID, metadata2.GUID, nil}
						l = append(l, a)

						for i, v := range l {
							if v.Org_id == metadata.GUID && v.Space_id == metadata2.GUID && v.Type == "space" {
								if len(l[i].Wires) == 0 {
									l[i].Wires = append(l[i].Wires, []string{metadata3.GUID})
								}else{
									l[i].Wires[0] = append(l[i].Wires[0], metadata3.GUID)
								}
							}
						}


					}
				}
			}
		}
	}





	//fmt.Println(l)

	//b := new(bytes.Buffer)
	//json.NewEncoder(b).Encode(ll)
	//
	//fmt.Println(b)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(l); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}


}

func (s *Server) ListConnectSpace(w http.ResponseWriter, r *http.Request) {

	orgData := s.GetOrg()
	spaceData := s.GetSpace()
	appData := s.GetApp()
	serviceData := s.GetServiceInstance()


	spaceid := r.FormValue(":spaceid")

	//fmt.Println(spaceid)

	l := []connNode{}

	iX := 550 //192
	iY := 250 //172
	iZ := 65535
	org := []string{}

	for _, value := range orgData.Resources {
		metadata := value.Metadata
		entity := value.Entity

		o := connNode{metadata.GUID, "org", entity.Name, entity.Status, iX, iY, iZ, nil, metadata.GUID, "", nil}
		l = append(l, o)

		for i, value2 := range spaceData.Resources {
			metadata2 := value2.Metadata
			entity2 := value2.Entity


			if metadata.GUID == entity2.Organization_guid && metadata2.GUID == spaceid {
				org = append(org, metadata.GUID)
				s := connNode{metadata2.GUID, "space", entity2.Name, "Active", iX+(i+150), iY, iZ, nil, metadata.GUID, metadata2.GUID, nil}
				l = append(l, s)

				for i, v := range l {
					if v.Org_id == metadata.GUID && v.Type == "org" {
						if len(l[i].Wires) == 0 {
							l[i].Wires = append(l[i].Wires, []string{metadata2.GUID})
						}else{
							l[i].Wires[0] = append(l[i].Wires[0], metadata2.GUID)
						}
					}
				}

				for j, value3 := range appData.Resources {
					metadata3 := value3.Metadata
					entity3 := value3.Entity


					if metadata2.GUID == entity3.Space_guid {
						a := connNode{metadata3.GUID, "app", entity3.Name, entity3.State, iX+(i+1+150)+300, iY+(j*50), iZ, nil, metadata.GUID, metadata2.GUID, nil}
						l = append(l, a)

						for i, v := range l {
							if v.Org_id == metadata.GUID && v.Space_id == metadata2.GUID && v.Type == "space" {
								if len(l[i].Wires) == 0 {
									l[i].Wires = append(l[i].Wires, []string{metadata3.GUID})
								}else{
									l[i].Wires[0] = append(l[i].Wires[0], metadata3.GUID)
								}
							}
						}
					}
				}

				for j, value4 := range serviceData.Resources {
					metadata4 := value4.Metadata
					entity4 := value4.Entity
					if metadata2.GUID == entity4.Space_guid {
						var nodetype = "svc"
						if len(entity4.Tags) > 0 {
							nodetype = entity4.Tags[0]
						}
						a := connNode{metadata4.GUID, nodetype, entity4.Name, "Active", iX+(i+1+150)+500, iY+(j*50), iZ, nil, metadata.GUID, metadata2.GUID, nil}
						l = append(l, a)
					}
				}
			}
		}
	}


	// App 연결정보 설정
	accessData, error := s.GetAccess()

	if error == nil {
		for i, node := range l {
			for _, acc := range accessData.Policies {
				if acc.Source.ID == node.ID {
					if len(l[i].Wires) == 0 {
						l[i].Wires = append(l[i].Wires, []string{acc.Destination.ID})
						//l[i].WiresInfo = append(l[i].WiresInfo, []wireInfo{acc.Destination.ID, acc.Destination.Protocol, acc.Destination.Port})
					} else {
						l[i].Wires[0] = append(l[i].Wires[0], acc.Destination.ID)
						//l[i].WiresInfo[0] = append(l[i].WiresInfo[0], wireInfo{acc.Destination.ID, acc.Destination.Protocol, acc.Destination.Port})
					}
					l[i].WiresInfo = append(l[i].WiresInfo, wireInfo{acc.Destination.ID, acc.Destination.Protocol, acc.Destination.Port.Start, ""})
				}
			}
		}
	} else {
		fmt.Println(error)
	}


	// App-Service 연결정보 설정
	serviceBindData := s.GetServiceBinding()
	for i, node := range l {
		for _, bind := range serviceBindData.Resources {
			if bind.Entity.App_guid == node.ID {
				if len(l[i].Wires) == 0 {
					l[i].Wires = append(l[i].Wires, []string{bind.Entity.Service_instance_guid})
				} else {
					l[i].Wires[0] = append(l[i].Wires[0], bind.Entity.Service_instance_guid)
				}
				l[i].WiresInfo = append(l[i].WiresInfo, wireInfo{"", "", 0, bind.Metadata.GUID})
			}
		}
	}


	// 해당 space 와 연결된 org만 추출.
	ll := []connNode{}
	exist := false
	for _, node := range l {
		for _, or := range org {
			if or == node.Org_id {
				exist = true;
			}
		}
		if exist == true {
			ll = append(ll, node)
		}
		exist = false;
	}



	//fmt.Println(l)

	//b := new(bytes.Buffer)
	//json.NewEncoder(b).Encode(ll)
	//
	//fmt.Println(b)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ll); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}


}

func (s *Server) userSpaces(userid string) ([]string, error) {
	logger := s.logger.Session("micro_server")
	logger.Debug("userSpaces")

	u := []string{}

	cf, err := s.uaa.CfClient()
	if err != nil {
		logger.Error("failed cf client", err)
		return u, err
	}

	// space 권한 조회
	users, err := cf.ListUsers()
	if err != nil {
		logger.Error("failed cf ListUsers", err)
		return u, err
	}

	user := users.GetUserByUsername(userid)
	spaces, err := cf.ListUserSpaces(user.Guid)
	if err != nil {
		logger.Error("failed cf ListUserSpaces", err)
		return u, err
	}


	for _, space := range spaces {
		u = append(u, space.Guid)
	}

	return u, nil
}

func (s *Server) CreateConnect(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	//fmt.Println(req.PostForm)

	p := Policies{}

	for k, v := range r.PostForm {
		if len(v) > 0 {
			if k == "source" {
				p.Source.ID = v[0]
			}
			if k == "target" {
				p.Destination.ID = v[0]
			}
			if k == "port" {
				if s, err := strconv.Atoi(v[0]); err == nil {
					p.Destination.Port.Start = s
				}
			}
			if k == "protocol" {
				p.Destination.Protocol = v[0]
			}
		}
	}

	pp := []Policies{}
	pp = append(pp, p)

	m := map[string][]Policies{"policies" : pp}

	fmt.Println(m, len(m["policies"]))

	err := s.CreateAccess(m)

	w.WriteHeader(http.StatusOK)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) DeleteConnect(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	//fmt.Println(req.PostForm)

	p := Policies{}

	for k, v := range r.PostForm {
		if len(v) > 0 {
			if k == "source" {
				p.Source.ID = v[0]
			}
			if k == "target" {
				p.Destination.ID = v[0]
			}
			if k == "port" {
				if s, err := strconv.Atoi(v[0]); err == nil {
					p.Destination.Port.Start = s
				}
			}
			if k == "protocol" {
				p.Destination.Protocol = v[0]
			}
		}
	}

	pp := []Policies{}
	pp = append(pp, p)

	m := map[string][]Policies{"policies" : pp}

	fmt.Println(m, len(m["policies"]))

	err := s.DeleteAccess(m)

	w.WriteHeader(http.StatusOK)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) ListConnectServiceSpace(w http.ResponseWriter, r *http.Request) {

	spaceid := r.FormValue(":spaceid")

	sr := ServiceResponse{}

	sv := s.GetServiceInstance()

	for _, value := range sv.Resources {
		if spaceid == value.Entity.Space_guid {
			sr.Resources = append(sr.Resources, value)
		}
	}

	fmt.Println(sr)

	//b := new(bytes.Buffer)
	//json.NewEncoder(b).Encode(ll)
	//
	//fmt.Println(b)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sr); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}


}

func (s *Server) CreateConnectService(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	//fmt.Println(req.PostForm)

	b := BindingService{}

	for k, v := range r.PostForm {
		if len(v) > 0 {
			if k == "source" {
				b.App_guid = v[0]
			}
			if k == "target" {
				b.Service_instance_guid = v[0]
			}
		}
	}


	save, err := s.CreateBinding(b)

	if err != nil {
		//panic(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(save); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) DeleteConnectService(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	//fmt.Println(req.PostForm)

	var serv = ""

	for k, v := range r.PostForm {
		if len(v) > 0 {
			if k == "bind" {
				serv = v[0]
			}
		}
	}


	s.DeleteBinding(serv)

	w.WriteHeader(http.StatusOK)
	//if err != nil {
	//	http.Error(resp, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	//}
}

func (s *Server) ListServiceMarketplace (w http.ResponseWriter, r *http.Request) {

	route := fmt.Sprintf("/v2/services")
	q := r.FormValue("q")
	if q != "" {
		route = fmt.Sprintf("%v?q=%v", route, q)
	}
	var services domain.Services
	err := s.uaa.GetResources("GET", route, nil, &services)
	if err != nil {
		//panic(error)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response domain.Services
	for _, service := range services.Resources {
		q = fmt.Sprintf("service_guid:%v", service.Meta.Guid)
		//service.Entity.Service_plan_guid = s.ListServicePlan(q).Resources[0].Metadata.GUID
		var servicePlans []string
		result := s.ListServicePlan(q)
		for _, plan := range result.Resources {
			servicePlans = append(servicePlans, plan.Metadata.GUID)
		}
		service.Entity.ServicePlans = servicePlans
		response.Resources = append(response.Resources, service)
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) Login (w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro_server")
	logger.Debug("Login")

	var request domain.TokenRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := url.Values{}
	data.Set("grant_type", request.GrantType)
	data.Set("response_type", request.ResponseType)
	data.Set("username", request.Username)
	data.Set("password", request.Password)

	token, err := s.uaa.GetUaaToken(data)
	if err != nil {
		s.logger.Error("failed cf get auth token", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		b := []byte(err.Error())
		w.Write(b)
		return
	}
	session := domain.SessionManager.Load(r)
	session.PutString(w, domain.UAA_TOKEN_NAME, token.AccessToken)
	session.PutString(w, domain.USER_ID, request.Username)

	auth := s.checkAuth(request.Username)

	m := make(map[string]string)
	m["result"] = base64.StdEncoding.EncodeToString([]byte(auth))
	if err := json.NewEncoder(w).Encode(m); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) Logout (w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("micro_server")
	logger.Debug("Logout")

	session := domain.SessionManager.Load(r)
	session.Destroy(w)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	m := make(map[string]bool)
	m["result"] = true
	if err := json.NewEncoder(w).Encode(m); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) ListServicePlan (q string) ServiceResponse {
	route := fmt.Sprintf("/v2/service_plans")
	if q != "" {
		route = fmt.Sprintf("%v?q=%v", route, q)
	}
	var response ServiceResponse
	err := s.uaa.GetResources("GET", route, nil, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}
	return response
}

func (s *Server) GetServiceMarketplaceByGuid (guid string) ServiceResource {
	route := fmt.Sprintf("/v2/services/%s", guid)
	var response ServiceResource
	err := s.uaa.GetResources("GET", route, nil, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}
	return response
}

func (s *Server) ListConnectService (w http.ResponseWriter, r *http.Request) {

	route := fmt.Sprintf("/v2/service_instances")
	q := r.FormValue("q")
	if q != "" {
		route = fmt.Sprintf("%v?q=%v", route, q)
	}
	var response ServiceResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		http.Error(w, error.Error(), http.StatusInternalServerError)
		return
	}

	//fmt.Fprintln(resp, response)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

}

func (s *Server) GetServiceInstance () ServiceResponse {

	route := fmt.Sprintf("/v2/service_instances")
	var response ServiceResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return response
}

func (s *Server) GetServiceInstanceByGuid (guid string) ServiceResource {

	route := fmt.Sprintf("/v2/service_instances/%s", guid)
	var response ServiceResource
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return response
}

func (s *Server) GetServiceInstanceByQuery (q string) (ServiceResponse, error) {

	route := fmt.Sprintf("/v2/service_instances?q=%v", q)
	var response ServiceResponse
	err := s.uaa.GetResources("GET", route, nil, &response)

	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return response, err
}

func (s *Server) CreateServiceInstance (reqData interface{}) (ServiceResource, error) {

	route := fmt.Sprintf("/v2/service_instances?accepts_incomplete=true")
	var response ServiceResource
	error := s.uaa.GetResources("POST", route, reqData, &response)

	if error != nil {
		s.logger.Error(error.Error(), error)
	}

	return response, error
}


func (s *Server) GetServiceBinding () ServiceResponse {

	route := fmt.Sprintf("/v2/service_bindings")
	var response ServiceResponse
	error := s.uaa.GetResources("GET", route, nil, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return response
}



func (s *Server) CreateBinding (reqData interface{}) (ServiceResource, error) {

	route := fmt.Sprintf("/v2/service_bindings")
	var response ServiceResource
	error := s.uaa.GetResources("POST", route, reqData, &response)

	if error != nil {
		//panic(error)
		s.logger.Error(error.Error(), error)
	}

	return response, error
}

func (s *Server) DeleteBinding (reqData interface{})  {

	if v, ok := reqData.(string); ok {
		fmt.Println(v)
		route := fmt.Sprintf("/v2/service_bindings/"+v+"?")
		//var response AccessResponse
		error := s.uaa.GetResources("DELETE", route, nil, nil)

		if error != nil {
			//panic(error)
			s.logger.Error(error.Error(), error)
		}
	}

}

func (s *Server) GetAppSummary (appguid string, token *client.CF_TOKEN) AppSummary {

	summary := fmt.Sprintf("/v2/apps/%s/summary", appguid)
	var response AppSummary
	err := s.uaa.GetResourcesFromToken("GET", summary, nil, &response, token.AccessToken)

	if err != nil {
		//panic(error)
		s.logger.Error(err.Error(), err)
	}

	//fmt.Println(response)

	return  response
}

func (s *Server) CreateApp (reqData interface{}, token *client.CF_TOKEN) (domain.AppResource, error) {

	route := fmt.Sprintf("/v2/apps")
	var response domain.AppResource
	error := s.uaa.GetResourcesFromToken("POST", route, reqData, &response, token.AccessToken)

	if error != nil {
		s.logger.Error(error.Error(), error)
	}

	return response, error
}

func (s *Server) CopyAppBits (appGuid string, reqData interface{}) (domain.AppResource, error) {

	route := fmt.Sprintf("/v2/apps/%s/copy_bits", appGuid)
	var response domain.AppResource
	error := s.uaa.GetResources("POST", route, reqData, &response)

	if error != nil {
		s.logger.Error(error.Error(), error)
	}

	return response, error
}

func (s *Server) UpdateApp (appGuid string, reqData interface{}, token *client.CF_TOKEN) (domain.AppResource, error) {

	route := fmt.Sprintf("/v2/apps/%s", appGuid)
	var response domain.AppResource
	error := s.uaa.GetResourcesFromToken("PUT", route, reqData, &response, token.AccessToken)

	if error != nil {
		s.logger.Error(error.Error(), error)
	}

	return response, error
}

func (s *Server) GetAppByName (name string) (domain.Apps, error) {

	route := fmt.Sprintf("/v2/apps?q=name:%s", name)
	var response domain.Apps
	err := s.uaa.GetResources("GET", route, nil, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return  response, err
}

func (s *Server) ListSpaceDomains (spaceGuid string) (domain.Results, error) {

	route := fmt.Sprintf("/v2/spaces/%s/domains", spaceGuid)
	var response domain.Results
	err := s.uaa.GetResources("GET", route, nil, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return  response, err
}

func (s *Server) ListServiceBindingByApp(appGuid string) (cfclient.ServiceBindingsResponse, error) {

	route := fmt.Sprintf("/v2/apps/%s/service_bindings", appGuid)
	var serviceBindingsResp cfclient.ServiceBindingsResponse

	err := s.uaa.GetResources("GET", route, nil, &serviceBindingsResp)

	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return serviceBindingsResp, err
}

func (s *Server) DeleteServiceBindingByApp(appGuid string, serviceBindingGuid string) error {

	route := fmt.Sprintf("/v2/apps/%s/service_bindings/%s", appGuid, serviceBindingGuid)

	err := s.uaa.GetResources("DELETE", route, nil, nil)

	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return err
}

func (s *Server) DeleteServiceInstanceByGuid (guid string) error {

	route := fmt.Sprintf("/v2/service_instances/%s?recursive=true", guid)
	err := s.uaa.GetResources("DELETE", route, nil, nil)

	if err != nil {
		//panic(error)
		s.logger.Error(err.Error(), err)
	}

	return err
}
func (s *Server) CreateRoute (reqData domain.Route) (domain.RoutesResource, error) {

	var response domain.RoutesResource
	err := s.uaa.GetResources("POST", "/v2/routes", reqData, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return response, err
}

func (s *Server) AssociateRoute (appGuid string, spaceGuid string) (domain.RoutesResource, error) {

	route := fmt.Sprintf("/v2/apps/%s/routes/%s", appGuid, spaceGuid)
	var response domain.RoutesResource
	err := s.uaa.GetResources("PUT", route, nil, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return response, err
}

func (s *Server) ListRouteForApp (appGuid string) (domain.RoutesResponse, error) {

	route := fmt.Sprintf("/v2/apps/%s/routes", appGuid)
	var response domain.RoutesResponse
	err := s.uaa.GetResources("GET", route, nil, &response)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return response, err
}

func (s *Server) DeleteRoute (routeGuid string) error {

	route := fmt.Sprintf("/v2/routes/%s?recursive=true", routeGuid)


	err := s.uaa.GetResources("DELETE", route, nil, nil)
	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	return err
}

func (s *Server) DeleteServiceBinding(serviceInstanceGuid string) error {

	route := fmt.Sprintf("/v2/service_instances/%s/service_bindings", serviceInstanceGuid)
	var serviceBindingsResp cfclient.ServiceBindingsResponse

	err := s.uaa.GetResources("GET", route, nil, &serviceBindingsResp)

	if err != nil {
		s.logger.Error(err.Error(), err)
	}

	for _, sbr := range serviceBindingsResp.Resources {

		route := fmt.Sprintf("/v2/service_bindings/%s", sbr.Meta.Guid)


		err := s.uaa.GetResources("DELETE", route, nil, nil)
		if err != nil {
			s.logger.Error(err.Error(), err)
		}
	}

	return err
}

func (s *Server) IsSpaceAuth(spaceGuid string, sessionUserid string) (bool, error) {
	cf, err := s.uaa.CfClient()
	if err != nil {
		//w.WriteHeader(http.StatusInternalServerError)
		//json.NewEncoder(w).Encode(err)
		fmt.Println(err)
		return false, err
	}

	// Manager check
	managerCheck := false
	if user, err := cf.ListSpaceManagers(spaceGuid); err == nil {
		for _, u := range user {
			if u.Username == sessionUserid {
				managerCheck = true
			}
		}
	}

	if !managerCheck {
		//logger.Error("failed ListSpaceManagers", err)
		//w.WriteHeader(http.StatusInternalServerError)
		//w.Write([]byte("not authorized"))
		return false, nil
	}

	return true, nil
}

func (s *Server) IsOrgAuth(orgGuid string, sessionUserid string) (bool, error) {
	cf, err := s.uaa.CfClient()
	if err != nil {
		//w.WriteHeader(http.StatusInternalServerError)
		//json.NewEncoder(w).Encode(err)
		fmt.Println(err)
		return false, err
	}

	// Manager check
	managerCheck := false
	if user, err := cf.ListOrgManagers(orgGuid); err == nil {
		for _, u := range user {
			if u.Username == sessionUserid {
				managerCheck = true
			}
		}
	}

	if !managerCheck {
		//logger.Error("failed ListSpaceManagers", err)
		//w.WriteHeader(http.StatusInternalServerError)
		//w.Write([]byte("not authorized"))
		return false, nil
	}

	return true, nil
}

func (s *Server) RestageApp (appGuid string, token *client.CF_TOKEN) (domain.AppResource, error) {

	route := fmt.Sprintf("/v2/apps/%s/restage", appGuid)
	var response domain.AppResource
	error := s.uaa.GetResourcesFromToken("POST", route, nil, &response, token.AccessToken)

	if error != nil {
		s.logger.Error(error.Error(), error)
	}

	return response, error
}

func (s *Server) IsOrgManageAuth(orgGuid string, sessionUserid string) (bool, error) {
	cf, err := s.uaa.CfClient()
	if err != nil {
		//w.WriteHeader(http.StatusInternalServerError)
		//json.NewEncoder(w).Encode(err)
		fmt.Println(err)
		return false, err
	}

	// Manager check
	managerCheck := false
	if user, err := cf.ListOrgManagers(orgGuid); err == nil {
		for _, u := range user {
			if u.Username == sessionUserid {
				managerCheck = true
			}
		}
	}

	if !managerCheck {
		//logger.Error("failed ListSpaceManagers", err)
		//w.WriteHeader(http.StatusInternalServerError)
		//w.Write([]byte("not authorized"))
		return false, nil
	}

	return true, nil
}

func (s *Server) checkAuth (userid string) string {
	logger := s.logger.Session("micro_server")
	logger.Debug("CheckAuth")

	orgData := s.GetOrg()
	for _, value := range orgData.Resources {
		metadata := value.Metadata
		if isAuth, err := s.IsOrgManageAuth(metadata.GUID, userid); err == nil {
			if isAuth {
				return "MANAGER"
			}
		}

	}

	return "MEMBER"

}
