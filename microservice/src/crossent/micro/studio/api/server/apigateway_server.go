package server

import (
	"bytes"
	"code.cloudfoundry.org/lager"
	"crossent/micro/studio/domain"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/containous/traefik/types"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) ListMicroserviceApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("ListMicroserviceApi")

	offset := 0
	if ioffset, err := strconv.Atoi(r.FormValue("offset")); err == nil {
		offset = ioffset
	}
	name := r.URL.Query().Get("name")

	// 세션 사용자 아이디
	//session := domain.SessionManager.Load(r)
	//userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	//spaces, _ := s.userSpaces(userid)
	org := r.FormValue("orgguid")

	views, err := s.repositoryFactory.Apigateway().ListMicroserviceApi(offset, name, org)


	if err != nil {
		logger.Error("failed ListMicroserviceApi", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(views)


}

/*
func (s *Server) PutMicroserviceApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("PutMicroserviceApi")

	// 세션 사용자 아이디
	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	//spaces, _ := s.userSpaces(userid)

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)

	var view domain.MicroApi
	err := json.NewDecoder(r.Body).Decode(&view)
	if err != nil {
		logger.Error("Decode err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	view.Userid = userid


	view.ID = idint
	//view.Swagger = string(body)
	err = s.repositoryFactory.Apigateway().SaveMicroserviceApi(view)

	if err != nil {
		logger.Error("failed PutMicroserviceApi", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")


}
*/

func (s *Server) CreateMicroserviceApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("CreateMicroserviceApi")

	// 세션 사용자 아이디
	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	//spaces, _ := s.userSpaces(userid)

	var microApi domain.MicroApi
	err := json.NewDecoder(r.Body).Decode(&microApi)
	if err != nil {
		logger.Error("Decode err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	microApi.Userid = userid

	// auth check
	if isAuth, err := s.IsOrgAuth(microApi.OrgGuid, userid); err == nil {
		if isAuth == false {
			logger.Debug("not authorized")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("not authorized"))
			return
		}
	}


	microApiRule, err := s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
	if err != nil {
		logger.Error("GetMicroserviceApiRule err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	view, err := s.repositoryFactory.View().GetMicroservice(microApi.MicroId)
	if err != nil {
		logger.Error("GetMicroservice err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if strings.HasPrefix(view.Url, "http") == false {
		view.Url = fmt.Sprintf("http://%s", view.Url)
	}

	var maxconn int64 =  10
	var period time.Duration = 3 * time.Second
	var average int64 = 5
	var burst int64 = 10
	headers := map[string]string{}
	whitelist := []string{}

	if microApi.MaxConn != "" {
		maxconn, _ =  strconv.ParseInt(microApi.MaxConn, 10, 64)
	}

	if microApi.Period != "" {
		p, _ := strconv.Atoi(microApi.Period)
		period =  time.Duration(p) * time.Second
	}

	if microApi.Average != "" {
		average, _ =  strconv.ParseInt(microApi.Average, 10, 64)
	}

	if microApi.Burst != "" {
		burst, _ =  strconv.ParseInt(microApi.Burst, 10, 64)
	}

	if microApi.Headers != nil {
		for _, header := range microApi.Headers {
			headers[header.Key] = header.Value
		}
	}

	if microApi.WhiteList != "" {
		whitelist = strings.Split(microApi.WhiteList, ",")
		for i, w := range whitelist {
			whitelist[i] = strings.TrimSpace(w)
		}
	}

	path := fmt.Sprintf("PathPrefix:%s", microApi.Path)
	pathStrip := ""
	method := ""
	host := ""

	if microApi.PathStrip == "Y" {
		pathStrip = fmt.Sprintf(";PathPrefixStrip:%s", microApi.Path)
	}

	if microApi.Method != "" {
		method = fmt.Sprintf(";Method:%s", microApi.Method)
	}

	if microApi.Host != "" {
		host = fmt.Sprintf(";Host:%s", microApi.Host)
	}

	frontkey := microApi.Name
	backkey := fmt.Sprintf("backend_%s", microApi.Name)

	hash, err := hashBcrypt(s.uaa.TraefikPassword)
	if err != nil {
		logger.Error("hashBcrypt error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if microApiRule.Rule == "" || microApiRule.Rule == "{}"{
		config := types.Configuration{
			Frontends: map[string]*types.Frontend{
				frontkey: {
					EntryPoints: []string{"http"},
					Routes: map[string]types.Route{
						"routes1" : {
							Rule: fmt.Sprintf("%s%s%s%s", path, pathStrip, method, host),
						},
					},
					Backend:     backkey,
					WhiteList: &types.WhiteList{
						SourceRange: whitelist,
					},
					Headers: &types.Headers{
						CustomRequestHeaders: headers,
					},
					BasicAuth: []string{ fmt.Sprintf("%s:%s", s.uaa.TraefikUser, hash) },
					RateLimit: &types.RateLimit{
						ExtractorFunc: "client.ip",
						RateSet: map[string]*types.Rate{
							"rateset1": {
								Period: period,
								Average: average,
								Burst: burst,
							},
						},
					},
				},
			},
			Backends: map[string]*types.Backend{
				backkey: {
					Servers: map[string]types.Server{
						"server1": {
							URL: view.Url,
						},
					},
					//LoadBalancer: &types.LoadBalancer{
					//	Method: lbMethod,
					//},
					//HealthCheck: healthCheck,
					MaxConn: &types.MaxConn{
						ExtractorFunc: "client.ip",
						Amount: maxconn,
					},
				},
			},
			//TLS: []*tls.Configuration{
			//	{
			//		Certificate: &tls.Certificate{
			//			CertFile: localhostCert,
			//			KeyFile:  localhostKey,
			//		},
			//		EntryPoints: []string{"http"},
			//	},
			//},
		}

		jsonConfig, err := json.Marshal(config)
		if err != nil {
			logger.Error("json marshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		microApi.Rule = string(jsonConfig)

	} else {
		var config types.Configuration

		err := json.Unmarshal([]byte(microApiRule.Rule), &config)
		if err != nil {
			logger.Error("json Unmarshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var basicAuth []string
		if config.Frontends[frontkey] != nil {
			//auths := config.Frontends[frontkey].BasicAuth
			//if len(auths) == 0 {
				config.Frontends[frontkey].BasicAuth = append(config.Frontends[frontkey].BasicAuth, fmt.Sprintf("%s:%s", s.uaa.TraefikUser, hash))
				basicAuth = config.Frontends[frontkey].BasicAuth
			//}

		} else {
			basicAuth = []string{ fmt.Sprintf("%s:%s", s.uaa.TraefikUser, hash) }
		}
		config.Frontends[frontkey] = &types.Frontend{
				EntryPoints: []string{"http"},
				Routes: map[string]types.Route{
					"routes1" : {
						Rule: fmt.Sprintf("%s%s%s", path, pathStrip, method),
					},
				},
				Backend:     backkey,
				WhiteList: &types.WhiteList{
					SourceRange: whitelist,
				},
				Headers: &types.Headers{
					CustomRequestHeaders: headers,
				},
				BasicAuth: basicAuth, //config.Frontends[frontkey].BasicAuth,
				RateLimit: &types.RateLimit{
					ExtractorFunc: "client.ip",
					RateSet: map[string]*types.Rate{
						"rateset1": {
							Period: period,
							Average: average,
							Burst: burst,
						},
					},
				},
		}

		config.Backends[backkey] = &types.Backend{
				Servers: map[string]types.Server{
					"server1": {
						URL: view.Url,
					},
				},
				MaxConn: &types.MaxConn{
					ExtractorFunc: "client.ip",
					Amount: maxconn,
				},
		}

		jsonConfig, err := json.Marshal(config)
		if err != nil {
			logger.Error("json marshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		microApi.Rule = string(jsonConfig)
	}


	if err = s.traefikApiPut(microApi.Rule, logger); err != nil {
		logger.Error("traefikApiPut err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// traefik rule save
	//go func(){
	//	bodyBytes, err := json.Marshal(microApi.Rule)
	//	if err != nil {
	//		logger.Error("traefik json marshal request body", err, lager.Data{"err": err.Error()})
	//		return
	//	}
		//reader := bytes.NewReader([]byte(microApi.Rule))
	//
	//	fmt.Println(string(bodyBytes))

		//request, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/providers/rest", s.uaa.TraefikApiUrl), reader)
		//if err != nil {
		//	logger.Error("http traefik request", err, lager.Data{"err": err.Error()})
		//	return
		//}
		//basicAuthToken := base64.StdEncoding.EncodeToString([]byte(s.uaa.TraefikUser + ":" + s.uaa.TraefikPassword))
		//request.Header.Set("Authorization", "Basic "+basicAuthToken)
		//request.Header.Set("Content-Type", "application/json;charset=UTF-8")
		//config := &tls.Config{InsecureSkipVerify: true}
		//tr := &http.Transport{TLSClientConfig: config}
		//client := &http.Client{Transport: tr}
		////client := &http.Client{}
		//
		//resp, err := client.Do(request)
		//if err != nil {
		//	logger.Error("http traefik response", err, lager.Data{"err": err.Error()})
		//	return
		//}
		//defer resp.Body.Close()
		//
		//logger.Debug("http traefik response code", lager.Data{"code": resp.StatusCode})
		//if code := resp.StatusCode; code < 200 || code > 299 {
		//	logger.Info("http new response code error", lager.Data{"code": code})
		//	w.WriteHeader(http.StatusInternalServerError)
		//	json.NewEncoder(w).Encode(resp.StatusCode)
		//	return
		//}


	//}()



	// JSON 바이트를 문자열로 변경
	//jsonString := string(jsonBytes)

	//config := th.BuildConfiguration(
	//	th.WithFrontends(th.WithFrontend("backend")),
	//	th.WithBackends(th.WithBackendNew("backend")),
	//)





	//configuration := new(types.Configuration)
	//if _, err := toml.Decode(content, configuration); err != nil {
	//	return nil, err
	//}
	//return configuration, nil

	//view.Swagger = string(body)
	err = s.repositoryFactory.Apigateway().SaveMicroserviceApi(microApi)

	if err != nil {
		logger.Error("failed CreateMicroserviceApi", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")


}

func (s *Server) ListMicroserviceApiHealth(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("ListMicroserviceApiHealth")

	//offset := 0
	//if ioffset, err := strconv.Atoi(r.FormValue("offset")); err == nil {
	//	offset = ioffset
	//}
	//name := r.URL.Query().Get("name")

	// 세션 사용자 아이디
	//session := domain.SessionManager.Load(r)
	//userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	//spaces, _ := s.userSpaces(userid)

	//views, err := s.repositoryFactory.Apigateway().ListMicroserviceApi(offset, name, "")

	//
	//if err != nil {
	//	logger.Error("failed ListMicroserviceApi", err)
	//	w.WriteHeader(http.StatusInternalServerError)
	//	json.NewEncoder(w).Encode(err)
	//	return
	//}

	var reader io.Reader
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/health", s.uaa.TraefikApiUrl), reader)
	if err != nil {
		logger.Error("http traefik request", err, lager.Data{"err": err.Error()})
		return
	}
	basicAuthToken := base64.StdEncoding.EncodeToString([]byte(s.uaa.TraefikUser + ":" + s.uaa.TraefikPassword))
	request.Header.Set("Authorization", "Basic "+basicAuthToken)
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	config := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}
	//client := &http.Client{}

	resp, err := client.Do(request)
	if err != nil {
		logger.Error("http traefik response", err, lager.Data{"err": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("http traefik response body", err, lager.Data{"err": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resp.StatusCode > 299 {
		var errDescription string

		var cfErr domain.CloudFoundryErr
		err = json.Unmarshal(bytes, &cfErr)
		if err != nil {
			errDescription = string(bytes)
		} else {
			errDescription = cfErr.Description
		}

		logger.Error("failed CreateMicroserviceApi", err)
		http.Error(w, errDescription, http.StatusInternalServerError)
		return

		//return fmt.Errorf(`{"httpStatusCode": %v, "cfErrorCode": "%s", "Message": "%s"}`, resp.StatusCode, cfErr.ErrorCode, errDescription)
	}

	//logger.Debug("http traefik response code", lager.Data{"code": resp.StatusCode})
	//if code := resp.StatusCode; code < 200 || code > 299 {
	//	logger.Info("http new response code error", lager.Data{"code": code})
	//	w.WriteHeader(http.StatusInternalServerError)
	//	json.NewEncoder(w).Encode(resp.StatusCode)
	//	return
	//}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)


}

func (s *Server) GetMicroserviceApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("GetMicroserviceApi")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	microApi, err := s.repositoryFactory.Apigateway().GetMicroserviceApi(idint)

	if err != nil {
		logger.Error("GetMicroserviceApi err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(microApi)
}

func (s *Server) GetMicroserviceApiUserList(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("GetMicroserviceApiUserList")

	id := r.FormValue(":id")
	idint, _ := strconv.Atoi(id)

	microid := r.URL.Query().Get("microId")
	microidint, _ := strconv.Atoi(microid)

	microApi, err := s.repositoryFactory.Apigateway().GetMicroserviceAppApi(microidint, idint)

	if err == nil {
		logger.Debug(microApi.UserName)
	}
}


func (s *Server) GetMicroserviceApiRule(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("GetMicroserviceApiRule")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	microApi, err := s.repositoryFactory.Apigateway().GetMicroserviceApi(idint)

	if err != nil {
		logger.Error("GetMicroserviceApi err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// auth check
	session := domain.SessionManager.Load(r)
	userid, err := session.GetString(domain.USER_ID)
	if err != nil {
		s.logger.Error("failed session auth token", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if microApi.Userid != userid {
		if isAuth, err := s.IsOrgAuth(microApi.OrgGuid, userid); err == nil {
			if isAuth == false {
				logger.Debug("not authorized")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("not authorized"))
				return
			}
		}
	}

	microApiRule, err := s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
	if err != nil {
		logger.Error("GetMicroserviceApiRule err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if microApiRule.Rule != "" {
		var config types.Configuration

		err = json.Unmarshal([]byte(microApiRule.Rule), &config)
		if err != nil {
			logger.Error("json Unmarshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		frontend := config.Frontends[microApi.Name]
		backend := config.Backends[fmt.Sprintf("backend_%s", microApi.Name)]

		if backend != nil && backend.MaxConn != nil {
			microApi.MaxConn = strconv.FormatInt(backend.MaxConn.Amount, 10)
		}
		microApi.PathStrip = "N"

		if frontend != nil {
			rule := frontend.Routes["routes1"].Rule
			rules := strings.Split(rule, ";")
			for _, r := range rules {
				if strings.HasPrefix(r, "PathPrefix:") {
					microApi.Path = strings.Replace(r, "PathPrefix:", "", 1)
				}
				if strings.HasPrefix(r, "PathPrefixStrip:") {
					microApi.PathStrip = "Y"
				}
				if strings.HasPrefix(r, "Method:") {
					microApi.Method = strings.Replace(r, "Method:", "", 1)
				}
				if strings.HasPrefix(r, "Host:") {
					microApi.Host = strings.Replace(r, "Host:", "", 1)
				}
			}
			microApi.WhiteList = strings.Join(frontend.WhiteList.SourceRange, ",")
			if frontend.RateLimit.RateSet["rateset1"] != nil {
				microApi.Period = fmt.Sprintf("%.f", frontend.RateLimit.RateSet["rateset1"].Period.Seconds())
				microApi.Average = strconv.FormatInt(frontend.RateLimit.RateSet["rateset1"].Average, 10)
				microApi.Burst = strconv.FormatInt(frontend.RateLimit.RateSet["rateset1"].Burst, 10)
			}

			headers := frontend.Headers.CustomRequestHeaders
			for key, val := range headers {
				microApi.Headers = append(microApi.Headers, domain.HeaderKeyValue{Key: key, Value: val})
			}
		}
	}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(microApi)



}

func (s *Server) GetMicroserviceApiSwagger(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("GetMicroserviceApiSwagger")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	view, err := s.repositoryFactory.View().GetMicroservice(idint)

	/*
	//swagger의 경우에는 모든 사용자가 API를 확인할 수 있어야함.
	if b := s.access(r, view.SpaceGuid); !b {
		logger.Error("no auth", fmt.Errorf("no auth"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no auth"))
		return
	}
	*/


	if err != nil {
		logger.Error("failed GetMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(view.Swagger)
	w.Write([]byte(view.Swagger))


}

func (s *Server) SaveMicroserviceApiSwagger(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("SaveMicroserviceApiSwagger")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	view, err := s.repositoryFactory.View().GetMicroservice(idint)

	if b := s.access(r, view.SpaceGuid); !b {
		logger.Error("no auth", fmt.Errorf("no auth"))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no auth"))
		return
	}

	if err != nil {
		logger.Error("failed GetMicroservice", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	//if view.Swagger == "" {
		logger.Info("swagger api request", lager.Data{"url": view.Url})
		resp, err := http.Get(fmt.Sprintf("http://%s/v2/api-docs", view.Url))
		if err != nil {
			logger.Error("failed http get api-docs", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			logger.Error("failed http get api-docs response", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err)
			return
		}

		if code := resp.StatusCode; code < 200 || code > 299 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(resp.StatusCode)
			return
		} else {

			view.ID = idint
			view.Swagger = string(body)
			err = s.repositoryFactory.Apigateway().SaveMicroserviceSwaggerApi(view)

			if err != nil {
				logger.Error("failed SaveMicroserviceApi", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}
		}
	//}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(view.Swagger)
	//w.Write([]byte("ok"))


}

func (s *Server) ListMicroserviceAppApiByMicroID(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("ListMicroserviceAppApiByMicroID")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	microApis, err := s.repositoryFactory.Apigateway().ListMicroserviceAppApiByMicroID(idint)

	if err != nil {
		logger.Error("GetMicroserviceApi err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(microApis)
}

func (s *Server) ListMicroserviceAppApiByApiID(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("ListMicroserviceAppApiByApiID")

	apiId := r.FormValue(":id")

	apiIdInt, _ := strconv.Atoi(apiId)
	microApis, err := s.repositoryFactory.Apigateway().ListMicroserviceAppApiByApiID(apiIdInt)

	if err != nil {
		logger.Error("GetMicroserviceApi err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(microApis)

}


func (s *Server) GetMicroserviceNameCheck(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("GetMicroserviceNameCheck")

	name := r.URL.Query().Get("name")

	result, err := s.repositoryFactory.Apigateway().GetMicroserviceNameCheck(name)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	if result == false || err != nil {
		w.Write([]byte(`{"result":"ng"}`))
		return
	}


	w.Write([]byte(`{"result":"ok"}`))
	//json.NewEncoder(w).Encode(microApi)

}

func (s *Server) GetMicroservicePathCheck(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("GetMicroserviceNameCheck")

	path := r.URL.Query().Get("path")

	result, err := s.repositoryFactory.Apigateway().GetMicroservicePathCheck(path)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	if result == false || err != nil {
		w.Write([]byte(`{"result":"ng"}`))
		return
	}


	w.Write([]byte(`{"result":"ok"}`))
	//json.NewEncoder(w).Encode(microApi)

}

func (s *Server) UpdateMicroserviceAppApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("UpdateMicroserviceAppApi")

	var microApi domain.MicroApi
	err := json.NewDecoder(r.Body).Decode(&microApi)
	if err != nil {
		logger.Error("Decode err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// path가 존재하지 않으면, 새로 api 추가하는 경우
	if strings.Contains(microApi.Path, "/") {
		microApiRule, err := s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
		if err != nil {
			logger.Error("GetMicroserviceApiRule err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		appApi, err := s.repositoryFactory.Apigateway().GetMicroserviceAppApi(microApi.MicroId, microApi.ID)
		if microApiRule.Rule != "" {
			var config types.Configuration
			// apiRule을 저장
			err = json.Unmarshal([]byte(microApiRule.Rule), &config)
			if err != nil {
				logger.Error("json Unmarshal err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			frontend := config.Frontends[microApi.Name]
			if frontend != nil {
				hash, err := hashBcrypt(microApi.Userpassword)
				if err != nil {
					logger.Error("hashBcrypt error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				for index, auth := range frontend.BasicAuth {
					user := strings.Split(auth, ":")
					if user[0] == appApi.UserName {
						frontend.BasicAuth = append(frontend.BasicAuth[:index], frontend.BasicAuth[index+1:]...)
						frontend.BasicAuth = append(frontend.BasicAuth, fmt.Sprintf("%s:%s", microApi.Username, hash))
					}
				}

				jsonConfig, err := json.Marshal(config)
				if err != nil {
					logger.Error("json marshal err", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				microApi.Rule = string(jsonConfig)
				if err = s.traefikApiPut(microApi.Rule, logger); err != nil {
					logger.Error("traefikApiPut err", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = s.repositoryFactory.Apigateway().SaveMicroserviceApiRule(microApi)
				if err != nil {
					logger.Error("failed CreateMicroserviceApi", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err)
					return
				}
			}
		}
	} else {
		logger.Error("There is no path while update is in progress.", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(microApi)
}

func (s *Server) CreateMicroserviceAppApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("CreateMicroserviceAppApi")

	//id := r.FormValue(":id")

	//idint, _ := strconv.Atoi(id)
	//view, err := s.repositoryFactory.View().GetMicroservice(idint)

	//if b := s.access(r, view.SpaceGuid); !b {
	//	logger.Error("no auth", fmt.Errorf("no auth"))
	//	w.WriteHeader(http.StatusInternalServerError)
	//	w.Write([]byte("no auth"))
	//	return
	//}

	//if err != nil {
	//	logger.Error("failed GetMicroservice", err)
	//	w.WriteHeader(http.StatusInternalServerError)
	//	json.NewEncoder(w).Encode(err)
	//	return
	//}


	var microApi domain.MicroApi
	err := json.NewDecoder(r.Body).Decode(&microApi)
	if err != nil {
		logger.Error("Decode err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := s.repositoryFactory.Apigateway().CreateMicroserviceAppApi(microApi)

	if err != nil {
		logger.Error("failed SaveMicroserviceAppApi", err)
		w.WriteHeader(http.StatusInternalServerError)
		//json.NewEncoder(w).Encode(err.Error())
		w.Write([]byte(err.Error()))
		return
	}

	if (result) {
		microApiRule, err := s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
		if err != nil {
			logger.Error("GetMicroserviceApiRule err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if microApiRule.Rule != "" {
			var config types.Configuration
			// apiRule을 저장
			err = json.Unmarshal([]byte(microApiRule.Rule), &config)
			if err != nil {
				logger.Error("json Unmarshal err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			frontend := config.Frontends[microApi.Name]

			if frontend != nil {
				//for _, auth := range frontend.BasicAuth {
				//	user := strings.Split(auth, ":")
				//	if user[0] == microApi.Username {
				//		logger.Debug("duplicate user")
				//		http.Error(w, "duplicate user", http.StatusInternalServerError)
				//		return
				//	}
				//}
				hash, err := hashBcrypt(microApi.Userpassword)
				if err != nil {
					logger.Error("hashBcrypt error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				frontend.BasicAuth = append(frontend.BasicAuth, fmt.Sprintf("%s:%s", microApi.Username, hash))

				jsonConfig, err := json.Marshal(config)
				if err != nil {
					logger.Error("json marshal err", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				microApi.Rule = string(jsonConfig)

				if err = s.traefikApiPut(microApi.Rule, logger); err != nil {
					logger.Error("traefikApiPut err", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = s.repositoryFactory.Apigateway().SaveMicroserviceApiRule(microApi)

				if err != nil {
					logger.Error("failed CreateMicroserviceApi", err)
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(err)
					return
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(microApi)
}




func (s *Server) ListMicroserviceFrontend(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("ListMicroserviceFrontend")

	// 세션 사용자 아이디
	session := domain.SessionManager.Load(r)
	userid, _ := session.GetString(domain.USER_ID)

	// 세션 사용자 속한 모든 space_guid
	spaces, _ := s.userSpaces(userid)

	views, err := s.repositoryFactory.Apigateway().ListMicroserviceFrontend(spaces)

	if err != nil {
		logger.Error("failed ListMicroserviceFrontend", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}

	newviews := []domain.View{}
	for _, view := range views {
		if view.Url != "" {
			urls := strings.Split(view.Url, ",")
			for _, url := range urls {
				newviews = append(newviews, domain.View{ID: view.ID, Url: url, Swagger: view.Swagger, Description: view.Description})
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newviews)

}


func (s *Server) DeleteMicroserviceApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("DeleteMicroserviceApi")

	id := r.FormValue(":id")

	idint, _ := strconv.Atoi(id)
	microApi, err := s.repositoryFactory.Apigateway().GetMicroserviceApi(idint)

	if err != nil {
		logger.Error("GetMicroserviceApi err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// auth check
	session := domain.SessionManager.Load(r)
	userid, err := session.GetString(domain.USER_ID)
	if err != nil {
		s.logger.Error("failed session auth token", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if microApi.Userid != userid {
		if isAuth, err := s.IsOrgAuth(microApi.OrgGuid, userid); err == nil {
			if isAuth == false {
				logger.Debug("not authorized")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("not authorized"))
				return
			}
		}
	}
	//view, err := s.repositoryFactory.View().GetMicroservice(idint)
	//
	//if b := s.access(r, view.SpaceGuid); !b {
	//	logger.Error("no auth", fmt.Errorf("no auth"))
	//	w.WriteHeader(http.StatusInternalServerError)
	//	w.Write([]byte("no auth"))
	//	return
	//}
	//
	//if err != nil {
	//	logger.Error("failed GetMicroservice", err)
	//	w.WriteHeader(http.StatusInternalServerError)
	//	json.NewEncoder(w).Encode(err)
	//	return
	//}


	microApiRule, err := s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
	if err != nil {
		logger.Error("GetMicroserviceApiRule err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	backupRule := ""

	if microApiRule.Rule != "" {
		var config types.Configuration

		err = json.Unmarshal([]byte(microApiRule.Rule), &config)
		if err != nil {
			logger.Error("json Unmarshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		backupRule = microApiRule.Rule

		delete(config.Frontends, microApi.Name)
		delete(config.Backends, fmt.Sprintf("backend_%s", microApi.Name))

		jsonConfig, err := json.Marshal(config)
		if err != nil {
			logger.Error("json marshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		microApi.Rule = string(jsonConfig)

		if err = s.traefikApiPut(microApi.Rule, logger); err != nil {
			logger.Error("traefikApiPut err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}



	err = s.repositoryFactory.Apigateway().DeleteMicroserviceApi(microApi, backupRule)

	if err != nil {
		logger.Error("failed DeleteMicroserviceApi", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(view.Swagger)
	//w.Write([]byte(view.Swagger))


}
/// 마이크로 서비스 수정 화면에서 삭제
func (s *Server) DeleteMicroserviceAppApi(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Session("apigateway")
	logger.Debug("DeleteMicroserviceAppApi")

	id := r.FormValue(":id")
	idint, _ := strconv.Atoi(id)

	microid := r.URL.Query().Get("microId")
	microidint, _ := strconv.Atoi(microid)

	microname := r.URL.Query().Get("microName")
	//view, err := s.repositoryFactory.View().GetMicroservice(idint)
	//
	//if b := s.access(r, view.SpaceGuid); !b {
	//	logger.Error("no auth", fmt.Errorf("no auth"))
	//	w.WriteHeader(http.StatusInternalServerError)
	//	w.Write([]byte("no auth"))
	//	return
	//}
	//
	//if err != nil {
	//	logger.Error("failed GetMicroservice", err)
	//	w.WriteHeader(http.StatusInternalServerError)
	//	json.NewEncoder(w).Encode(err)
	//	return
	//}

	// 20200131 -- s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
	// 20200131 -- s.repositoryFactory.Apigateway().SaveMicroserviceApiRule

	// 1. micro_api_rule에서 rule을 조회
	microApiRule, err := s.repositoryFactory.Apigateway().GetMicroserviceApiRule()
	if err != nil {
		logger.Error("GetMicroserviceApiRule err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(microApiRule.Active)
	// 2. micro_app_api 로 부터 username을 알아낸다.
	appApi, err := s.repositoryFactory.Apigateway().GetMicroserviceAppApi(microidint, idint)
	fmt.Println(appApi.Active)

	// 3. frontend.BasioAuth에서 id값에 맞는 걸 제거
	if microApiRule.Rule != "" {

		var config types.Configuration
		// apiRule을 저장
		err = json.Unmarshal([]byte(microApiRule.Rule), &config)
		if err != nil {
			logger.Error("json Unmarshal err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		frontend := config.Frontends[microname]

		indexCorrect :=0
		if frontend != nil {
			for index, auth := range frontend.BasicAuth {
				user := strings.Split(auth, ":")
				if user[0] == appApi.UserName {
			//		logger.Debug("duplicate user")
			//		http.Error(w, "duplicate user", http.StatusInternalServerError)
			//		return
					frontend.BasicAuth = append(frontend.BasicAuth[:(index-indexCorrect)], frontend.BasicAuth[(index-indexCorrect)+1:]...)
					indexCorrect++
				}
			}
			microApi := domain.MicroApi{}

			//frontend.BasicAuth = append(frontend.BasicAuth, fmt.Sprintf("%s:%s", microApi.Username, hash))

			jsonConfig, err := json.Marshal(config)
			if err != nil {
				logger.Error("json marshal err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			microApi.Rule = string(jsonConfig)

			if err = s.traefikApiPut(microApi.Rule, logger); err != nil {
				logger.Error("traefikApiPut err", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = s.repositoryFactory.Apigateway().SaveMicroserviceApiRule(microApi)

			if err != nil {
				logger.Error("failed CreateMicroserviceApi", err)
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err)
				return
			}

		}
	}


	err = s.repositoryFactory.Apigateway().DeleteMicroserviceAppApi(idint, microidint, appApi.UserName)

	if err != nil {
		logger.Error("failed DeleteMicroserviceApi", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(view.Swagger)
	//w.Write([]byte(view.Swagger))
}

func (s *Server) traefikApiPut(rule string, logger lager.Logger) error {
	reader := bytes.NewReader([]byte(rule))

	request, err := http.NewRequest("PUT", fmt.Sprintf("%s/api/providers/rest", s.uaa.TraefikApiUrl), reader)
	if err != nil {
		logger.Error("http traefik request", err, lager.Data{"err": err.Error()})
		return err
	}
	basicAuthToken := base64.StdEncoding.EncodeToString([]byte(s.uaa.TraefikUser + ":" + s.uaa.TraefikPassword))
	request.Header.Set("Authorization", "Basic "+basicAuthToken)
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	config := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}
	//client := &http.Client{}

	resp, err := client.Do(request)
	if err != nil {
		logger.Error("http traefik response", err, lager.Data{"err": err.Error()})
		return err
	}
	defer resp.Body.Close()

	logger.Debug("http traefik response code", lager.Data{"code": resp.StatusCode, "url": request.URL.String(), "rule": rule})
	if code := resp.StatusCode; code < 200 || code > 299 {
		logger.Info("http new response code error", lager.Data{"code": code})
		return fmt.Errorf(fmt.Sprintf("%s", resp.StatusCode))
	}
	return nil
}

func hashBcrypt(password string) (hash string, err error) {
	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	return string(passwordBytes), nil
}